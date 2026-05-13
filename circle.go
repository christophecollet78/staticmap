package main

import (
	"encoding/hex"
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"github.com/golang/geo/s2"
	colorful "github.com/lucasb-eyer/go-colorful"
	"github.com/pkg/errors"
)

type circle struct {
	pos    s2.LatLng
	color  color.Color
	fill   color.Color
	radius float64
	weight float64
}

func (c circle) String() string {
	sr, sg, sb, sa := c.color.RGBA()
	fr, fg, fb, fa := c.fill.RGBA()
	return fmt.Sprintf("%s|%.2f|%.2f|%d,%d,%d,%d|%d,%d,%d,%d",
		c.pos.String(),
		c.radius,
		c.weight,
		sr, sg, sb, sa,
		fr, fg, fb, fa,
	)
}

// parseColorValue accepts a named color (markerColors), a 0xRGB / 0xRRGGBB hex
// (no alpha, returned as color.RGBA via colorful) or a 0xRRGGBBAA hex (returned
// as color.NRGBA so the alpha is interpreted as non-premultiplied — required
// by the go-staticmaps fill rendering, otherwise color.RGBA would assume
// premultiplied components and produce a wrong tint).
func parseColorValue(v string) (color.Color, error) {
	if strings.HasPrefix(v, "0x") {
		h := strings.TrimPrefix(v, "0x")
		if len(h) == 8 { //nolint:mnd
			b, err := hex.DecodeString(h)
			if err != nil {
				return nil, errors.Wrapf(err, "parsing hex color %q", v)
			}
			return color.NRGBA{R: b[0], G: b[1], B: b[2], A: b[3]}, nil
		}
		c, err := colorful.Hex("#" + h)
		if err != nil {
			return nil, errors.Wrapf(err, "parsing hex color %q", v)
		}
		return c, nil
	}

	c, ok := markerColors[v]
	if !ok {
		return nil, errors.Errorf("bad color name %q", v)
	}
	return c, nil
}

func parseCircles(circles []string) ([]circle, error) {
	if circles == nil {
		return nil, nil
	}

	result := []circle{}

	for _, circleInformation := range circles {
		parts := strings.Split(circleInformation, "|")

		var (
			pos       s2.LatLng
			posSet    bool
			col       color.Color = markerColors["red"]
			fillCol   color.Color
			fillSet   bool
			radius    = 100.0 //nolint:mnd
			weight    = 3.0   //nolint:mnd
			parseErr  error
		)

		for _, p := range parts {
			switch {
			case strings.HasPrefix(p, "radius:"):
				v, err := strconv.ParseFloat(strings.TrimPrefix(p, "radius:"), 64)
				if err != nil {
					return nil, errors.Wrapf(err, "parsing radius %q", strings.TrimPrefix(p, "radius:"))
				}
				radius = v

			case strings.HasPrefix(p, "weight:"):
				v, err := strconv.ParseFloat(strings.TrimPrefix(p, "weight:"), 64)
				if err != nil {
					return nil, errors.Wrapf(err, "parsing weight %q", strings.TrimPrefix(p, "weight:"))
				}
				weight = v

			case strings.HasPrefix(p, "color:"):
				c, err := parseColorValue(strings.TrimPrefix(p, "color:"))
				if err != nil {
					return nil, errors.Wrap(err, "parsing color")
				}
				col = c

			case strings.HasPrefix(p, "fill:"):
				c, err := parseColorValue(strings.TrimPrefix(p, "fill:"))
				if err != nil {
					return nil, errors.Wrap(err, "parsing fill")
				}
				fillCol = c
				fillSet = true

			default:
				pos, parseErr = parseCoordinate(p)
				if parseErr != nil {
					return nil, errors.Errorf("unparsable chunk found in circle: %q", p)
				}
				posSet = true
			}
		}

		if !posSet {
			return nil, errors.New("circle is missing required lat,lon coordinates")
		}

		if !fillSet {
			r, g, b, _ := col.RGBA()
			fillCol = color.NRGBA{
				R: uint8(r >> 8), //nolint:mnd
				G: uint8(g >> 8), //nolint:mnd
				B: uint8(b >> 8), //nolint:mnd
				A: 64,            //nolint:mnd
			}
		}

		result = append(result, circle{
			pos:    pos,
			color:  col,
			fill:   fillCol,
			radius: radius,
			weight: weight,
		})
	}

	return result, nil
}
