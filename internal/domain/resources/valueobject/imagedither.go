package valueobject

import (
	"github.com/disintegration/gift"
	"github.com/makeworld-the-better-one/dither/v2"
	"image"
	"image/draw"
)

var _ gift.Filter = (*ditherFilter)(nil)

type ditherFilter struct {
	ditherer *dither.Ditherer
}

var ditherMethodsErrorDiffusion = map[string]dither.ErrorDiffusionMatrix{
	"atkinson":            dither.Atkinson,
	"burkes":              dither.Burkes,
	"falsefloydsteinberg": dither.FalseFloydSteinberg,
	"floydsteinberg":      dither.FloydSteinberg,
	"jarvisjudiceninke":   dither.JarvisJudiceNinke,
	"sierra":              dither.Sierra,
	"sierra2":             dither.Sierra2,
	"sierra2_4a":          dither.Sierra2_4A,
	"sierra3":             dither.Sierra3,
	"sierralite":          dither.SierraLite,
	"simple2d":            dither.Simple2D,
	"stevenpigeon":        dither.StevenPigeon,
	"stucki":              dither.Stucki,
	"tworowsierra":        dither.TwoRowSierra,
}

var ditherMethodsOrdered = map[string]dither.OrderedDitherMatrix{
	"clustereddot4x4":            dither.ClusteredDot4x4,
	"clustereddot6x6":            dither.ClusteredDot6x6,
	"clustereddot6x6_2":          dither.ClusteredDot6x6_2,
	"clustereddot6x6_3":          dither.ClusteredDot6x6_3,
	"clustereddot8x8":            dither.ClusteredDot8x8,
	"clustereddotdiagonal16x16":  dither.ClusteredDotDiagonal16x16,
	"clustereddotdiagonal6x6":    dither.ClusteredDotDiagonal6x6,
	"clustereddotdiagonal8x8":    dither.ClusteredDotDiagonal8x8,
	"clustereddotdiagonal8x8_2":  dither.ClusteredDotDiagonal8x8_2,
	"clustereddotdiagonal8x8_3":  dither.ClusteredDotDiagonal8x8_3,
	"clustereddothorizontalline": dither.ClusteredDotHorizontalLine,
	"clustereddotspiral5x5":      dither.ClusteredDotSpiral5x5,
	"clustereddotverticalline":   dither.ClusteredDotVerticalLine,
	"horizontal3x5":              dither.Horizontal3x5,
	"vertical5x3":                dither.Vertical5x3,
}

func (f ditherFilter) Draw(dst draw.Image, src image.Image, options *gift.Options) {
	gift.New().Draw(dst, f.ditherer.Dither(src))
}

func (f ditherFilter) Bounds(srcBounds image.Rectangle) image.Rectangle {
	return image.Rect(0, 0, srcBounds.Dx(), srcBounds.Dy())
}
