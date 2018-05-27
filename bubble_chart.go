package chart

import (
	"errors"
	"io"
	"math"
	"github.com/golang/freetype/truetype"
	"daill.de/go-chart/util"
	"fmt"
)

// BarChart is a chart that draws bars on a range.
type BubbleChart struct {
	Title      string
	TitleStyle Style

	ColorPalette ColorPalette

	Width  int
	Height int
	DPI    float64

	BubbleScale float64

	Background Style
	Canvas     Style

	XAxis XAxis
	YAxis YAxis

	Font        *truetype.Font
	defaultFont *truetype.Font

	Bubbles     []BubbleValue
	Elements []Renderable
}

// GetDPI returns the dpi for the chart.
func (bc BubbleChart) GetDPI() float64 {
	if bc.DPI == 0 {
		return DefaultDPI
	}
	return bc.DPI
}

// GetFont returns the text font.
func (bc BubbleChart) GetFont() *truetype.Font {
	if bc.Font == nil {
		return bc.defaultFont
	}
	return bc.Font
}

// GetWidth returns the chart width or the default value.
func (bc BubbleChart) GetWidth() int {
	if bc.Width == 0 {
		return DefaultChartWidth
	}
	return bc.Width
}

// GetHeight returns the chart height or the default value.
func (bc BubbleChart) GetHeight() int {
	if bc.Height == 0 {
		return DefaultChartHeight
	}
	return bc.Height
}

func (bc BubbleChart) getBubbleScale() float64 {
	if bc.BubbleScale == 0 {
		return 1
	}
	return bc.BubbleScale
}

// Render renders the chart with the given renderer to the given io.Writer.
func (bc BubbleChart) Render(rp RendererProvider, w io.Writer) error {
	if len(bc.Bubbles) == 0 {
		return errors.New("please provide at least one bubble")
	}

	r, err := rp(bc.GetWidth(), bc.GetHeight())
	if err != nil {
		return err
	}

	if bc.Font == nil {
		defaultFont, err := GetDefaultFont()
		if err != nil {
			return err
		}
		bc.defaultFont = defaultFont
	}
	r.SetDPI(bc.GetDPI())

	bc.drawBackground(r)

	var canvasBox Box
	var yt []Tick
	var yr Range
	var yf ValueFormatter

	var xt []Tick
	var xr Range
	var xf ValueFormatter

	canvasBox = bc.getDefaultCanvasBox()
	yr = bc.getYRanges()
	if yr.GetMax()-yr.GetMin() == 0 {
		// return fmt.Errorf("invalid data range; cannot be zero")
		v := yr.GetMax()
		if v > 0 {
			yr.SetMin(0)
		} else if  v < 0 {
			yr.SetMax(0)
		} else {
			yr.SetMax(0.5)
			yr.SetMin(0)
		}
	}
	yr = bc.setYRangeDomains(canvasBox, yr)
	yf = bc.getValueFormatters()

	xr = bc.getYRanges()
	xr = bc.setYRangeDomains(canvasBox, xr)
	xf = bc.getValueFormatters()

	if bc.hasAxes() {
		yt = bc.getYAxesTicks(r, yr, yf)
		canvasBox = bc.getAdjustedCanvasBox(r, canvasBox, yr, yt)
		yr = bc.setYRangeDomains(canvasBox, yr)

		xt = bc.getXAxesTicks(r, xr, xf)
		//canvasBox = bc.getAdjustedCanvasBox(r, canvasBox, xr, xt)
		xr = bc.setXRangeDomains(canvasBox, xr)
	}

	bc.drawCanvas(r, canvasBox)
	bc.drawBubbles(r, canvasBox, xr, yr)
	bc.drawXAxis(r, canvasBox, xr, xt)
	bc.drawYAxis(r, canvasBox, yr, yt)

	bc.drawTitle(r)
	for _, a := range bc.Elements {
		a(r, canvasBox, bc.styleDefaultsElements())
	}

	return r.Save(w)
}

func (bc BubbleChart) drawCanvas(r Renderer, canvasBox Box) {
	Draw.Box(r, canvasBox, bc.getCanvasStyle())
}

func (bc BubbleChart) getXRanges() Range {
	var xrange Range
	if bc.XAxis.Range != nil && !bc.XAxis.Range.IsZero() {
		xrange = bc.XAxis.Range
	} else {
		xrange = &ContinuousRange{}
	}

	if !xrange.IsZero() {
		return xrange
	}

	if len(bc.XAxis.Ticks) > 0 {
		tickMin, tickMax := math.MaxFloat64, -math.MaxFloat64
		for _, t := range bc.XAxis.Ticks {
			tickMin = math.Min(tickMin, t.Value)
			tickMax = math.Max(tickMax, t.Value)
		}
		xrange.SetMin(tickMin)
		xrange.SetMax(tickMax)
		return xrange
	}

	min, max := math.MaxFloat64, -math.MaxFloat64
	for _, b := range bc.Bubbles {
		min = math.Min(b.XVal, min)
		max = math.Max(b.XVal, max)
	}

	xrange.SetMin(min)
	xrange.SetMax(max)

	return xrange
}

func (bc BubbleChart) getYRanges() Range {
	var yrange Range
	if bc.YAxis.Range != nil && !bc.YAxis.Range.IsZero() {
		yrange = bc.YAxis.Range
	} else {
		yrange = &ContinuousRange{}
	}

	if !yrange.IsZero() {
		return yrange
	}

	if len(bc.YAxis.Ticks) > 0 {
		tickMin, tickMax := math.MaxFloat64, -math.MaxFloat64
		for _, t := range bc.YAxis.Ticks {
			tickMin = math.Min(tickMin, t.Value)
			tickMax = math.Max(tickMax, t.Value)
		}
		yrange.SetMin(tickMin)
		yrange.SetMax(tickMax)
		return yrange
	}

	min, max := math.MaxFloat64, -math.MaxFloat64
	for _, b := range bc.Bubbles {
		min = math.Min(b.YVal, min)
		max = math.Max(b.YVal, max)
	}

	yrange.SetMin(min)
	yrange.SetMax(max)

	return yrange
}

func (bc BubbleChart) drawBackground(r Renderer) {
	Draw.Box(r, Box{
		Right:  bc.GetWidth(),
		Bottom: bc.GetHeight(),
	}, bc.getBackgroundStyle())
}

func (bc BubbleChart) drawBubbles(r Renderer, canvasBox Box, xr, yr Range) {
	xoffset := canvasBox.Left
	yoffset := canvasBox.Bottom

	var bubbleBox Bubble
	var tb Box
	var text string
 	for index, bubble := range bc.Bubbles {
		bubbleBox = Bubble{
			MidPointX: xoffset+int(xr.Translate(bubble.XVal)),
			MidPointY: yoffset-int(yr.Translate(bubble.YVal)),
			Radius: int(bubble.Value.Value*bc.getBubbleScale()),
		}

		Draw.Circle(r, bubbleBox, bubble.Value.Style.InheritFrom(bc.styleDefaultsBar(index)))

		text = fmt.Sprintf("%v", bubble.Value.Value)
		tb = r.MeasureText(text)
		Draw.Text(r, text, xoffset+int(xr.Translate(bubble.XVal)), yoffset-int(yr.Translate(bubble.YVal))+(tb.Height())+int(bubble.Value.Value*bc.getBubbleScale()), bubble.Value.Style.InheritFrom(bc.styleDefaultsAxes()))
	}
}


func (bc BubbleChart) drawXAxis(r Renderer, canvasBox Box, xr Range, ticks []Tick) {

	if bc.XAxis.Style.Show {
		axisStyle := bc.XAxis.Style.InheritFrom(bc.styleDefaultsAxes())
		axisStyle.WriteToRenderer(r)

		r.MoveTo(canvasBox.Left, canvasBox.Bottom)
		r.LineTo(canvasBox.Right, canvasBox.Bottom)
		r.Stroke()

		r.MoveTo(canvasBox.Left, canvasBox.Bottom)
		r.LineTo(canvasBox.Left, canvasBox.Bottom+DefaultVerticalTickHeight)
		r.Stroke()


		var tx int
		var tb Box
		for _, t := range ticks {
			tx = canvasBox.Left + xr.Translate(t.Value)

			axisStyle.GetStrokeOptions().WriteToRenderer(r)
			r.MoveTo(tx, canvasBox.Bottom)
			r.LineTo(tx, canvasBox.Bottom+DefaultVerticalTickHeight)
			r.Stroke()

			axisStyle.GetTextOptions().WriteToRenderer(r)
			tb = r.MeasureText(t.Label)
			Draw.Text(r, t.Label, tx-(tb.Width()>>1), canvasBox.Bottom+DefaultXAxisMargin+10, axisStyle)
		}
		//cursor := canvasBox.Left
		//for index, bubble := range bc.Bubbles {
		//	barLabelBox := Box{
		//		Top:    canvasBox.Bottom + DefaultXAxisMargin,
		//		Left:   cursor,
		//		Right:  cursor + int(bubble.Value.Value),
		//		Bottom: bc.GetHeight(),
		//	}
		//
		//	if len(bubble.Value.Label) > 0 {
		//		Draw.TextWithin(r, bubble.Value.Label, barLabelBox, axisStyle)
		//	}
		//
		//	axisStyle.WriteToRenderer(r)
		//	if index < len(bc.Bubbles)-1 {
		//		r.MoveTo(barLabelBox.Right, canvasBox.Bottom)
		//		r.LineTo(barLabelBox.Right, canvasBox.Bottom+DefaultVerticalTickHeight)
		//		r.Stroke()
		//	}
		//	cursor += int(bubble.Value.Value*scale)
		//}
	}
}

func (bc BubbleChart) drawYAxis(r Renderer, canvasBox Box, yr Range, ticks []Tick) {
	if bc.YAxis.Style.Show {
		axisStyle := bc.YAxis.Style.InheritFrom(bc.styleDefaultsAxes())
		axisStyle.WriteToRenderer(r)

		r.MoveTo(canvasBox.Right, canvasBox.Top)
		r.LineTo(canvasBox.Right, canvasBox.Bottom)
		r.Stroke()

		r.MoveTo(canvasBox.Right, canvasBox.Bottom)
		r.LineTo(canvasBox.Right+DefaultHorizontalTickWidth, canvasBox.Bottom)
		r.Stroke()

		var ty int
		var tb Box
		for _, t := range ticks {
			ty = canvasBox.Bottom - yr.Translate(t.Value)

			axisStyle.GetStrokeOptions().WriteToRenderer(r)
			r.MoveTo(canvasBox.Right, ty)
			r.LineTo(canvasBox.Right+DefaultHorizontalTickWidth, ty)
			r.Stroke()

			axisStyle.GetTextOptions().WriteToRenderer(r)
			tb = r.MeasureText(t.Label)
			Draw.Text(r, t.Label, canvasBox.Right+DefaultYAxisMargin+5, ty+(tb.Height()>>1), axisStyle)
		}

	}
}

func (bc BubbleChart) drawTitle(r Renderer) {
	if len(bc.Title) > 0 && bc.TitleStyle.Show {
		r.SetFont(bc.TitleStyle.GetFont(bc.GetFont()))
		r.SetFontColor(bc.TitleStyle.GetFontColor(bc.GetColorPalette().TextColor()))
		titleFontSize := bc.TitleStyle.GetFontSize(bc.getTitleFontSize())
		r.SetFontSize(titleFontSize)

		textBox := r.MeasureText(bc.Title)

		textWidth := textBox.Width()
		textHeight := textBox.Height()

		titleX := (bc.GetWidth() >> 1) - (textWidth >> 1)
		titleY := bc.TitleStyle.Padding.GetTop(DefaultTitleTop) + textHeight

		r.Text(bc.Title, titleX, titleY)
	}
}

func (bc BubbleChart) getCanvasStyle() Style {
	return bc.Canvas.InheritFrom(bc.styleDefaultsCanvas())
}

func (bc BubbleChart) styleDefaultsCanvas() Style {
	return Style{
		FillColor:   bc.GetColorPalette().CanvasColor(),
		StrokeColor: bc.GetColorPalette().CanvasStrokeColor(),
		StrokeWidth: DefaultCanvasStrokeWidth,
	}
}

func (bc BubbleChart) hasAxes() bool {
	return bc.YAxis.Style.Show
}

func (bc BubbleChart) setYRangeDomains(canvasBox Box, yr Range) Range {
	yr.SetDomain(canvasBox.Height())
	return yr
}

func (bc BubbleChart) setXRangeDomains(canvasBox Box, xr Range) Range {
	xr.SetDomain(canvasBox.Width())
	return xr
}

func (bc BubbleChart) getDefaultCanvasBox() Box {
	return bc.box()
}

func (bc BubbleChart) getValueFormatters() ValueFormatter {
	if bc.YAxis.ValueFormatter != nil {
		return bc.YAxis.ValueFormatter
	}
	return FloatValueFormatter
}

func (bc BubbleChart) getXAxesTicks(r Renderer, xr Range, xf ValueFormatter) (xticks []Tick) {
	if bc.XAxis.Style.Show {
		xticks = bc.XAxis.GetTicks(r, xr, bc.styleDefaultsAxes(), xf)
	}
	return
}

func (bc BubbleChart) getYAxesTicks(r Renderer, yr Range, yf ValueFormatter) (yticks []Tick) {
	if bc.YAxis.Style.Show {
		yticks = bc.YAxis.GetTicks(r, yr, bc.styleDefaultsAxes(), yf)
	}
	return
}

//func (bc BubbleChart) calculateEffectiveBarSpacing(canvasBox Box) int {
//	totalWithBaseSpacing := bc.calculateTotalBarWidth(bc.GetBarWidth(), bc.GetBarSpacing())
//	//if totalWithBaseSpacing > canvasBox.Width() {
//	//	lessBarWidths := canvasBox.Width() - (len(bc.Bars) * bc.GetBarWidth())
//	//	if lessBarWidths > 0 {
//	//		return int(math.Ceil(float64(lessBarWidths) / float64(len(bc.Bars))))
//	//	}
//	//	return 0
//	//}
//	return 0
//}
//
//func (bc BubbleChart) calculateEffectiveBarWidth(canvasBox Box, spacing int) int {
//	//totalWithBaseWidth := bc.calculateTotalBarWidth(bc.GetBarWidth(), spacing)
//	//if totalWithBaseWidth > canvasBox.Width() {
//	//	totalLessBarSpacings := canvasBox.Width() - (len(bc.Bars) * spacing)
//	//	if totalLessBarSpacings > 0 {
//	//		return int(math.Ceil(float64(totalLessBarSpacings) / float64(len(bc.Bars))))
//	//	}
//	//	return 0
//	//}
//	return bc.calculateTotalBarWidth(bc.GetBarWidth(), spacing)
//}

func (bc BubbleChart) calculateTotalBarWidth() int {
	// max y value + radius + offset
	width := 0.0
	for _, bubble := range bc.Bubbles {
		width = math.Max(bubble.YVal+bubble.Value.Value, width)
	}
	return int(width)
}

func (bc BubbleChart) calculateScaledTotalWidth(canvasBox Box) int {
	return bc.calculateTotalBarWidth()
}

func (bc BubbleChart) getAdjustedCanvasBox(r Renderer, canvasBox Box, yrange Range, yticks []Tick) Box {
	axesOuterBox := canvasBox.Clone()

	totalWidth := bc.calculateScaledTotalWidth(canvasBox)

	if bc.XAxis.Style.Show {
		xaxisHeight := DefaultVerticalTickHeight

		axisStyle := bc.XAxis.Style.InheritFrom(bc.styleDefaultsAxes())
		axisStyle.WriteToRenderer(r)

		cursor := canvasBox.Left
		for _, bubble := range bc.Bubbles {
			if len(bubble.Value.Label) > 0 {
				barLabelBox := Box{
					Top:    canvasBox.Bottom + DefaultXAxisMargin,
					Left:   cursor,
					Right:  cursor + DefaultBarWidth,
					Bottom: bc.GetHeight(),
				}
				lines := Text.WrapFit(r, bubble.Value.Label, barLabelBox.Width(), axisStyle)
				linesBox := Text.MeasureLines(r, lines, axisStyle)

				xaxisHeight = util.Math.MinInt(linesBox.Height()+(2*DefaultXAxisMargin), xaxisHeight)
			}
		}

		xbox := Box{
			Top:    canvasBox.Top,
			Left:   canvasBox.Left,
			Right:  canvasBox.Left + totalWidth,
			Bottom: bc.GetHeight() - xaxisHeight,
		}

		axesOuterBox = axesOuterBox.Grow(xbox)
	}

	if bc.YAxis.Style.Show {
		axesBounds := bc.YAxis.Measure(r, canvasBox, yrange, bc.styleDefaultsAxes(), yticks)
		axesOuterBox = axesOuterBox.Grow(axesBounds)
	}

	return canvasBox.OuterConstrain(bc.box(), axesOuterBox)
}

// box returns the chart bounds as a box.
func (bc BubbleChart) box() Box {
	dpr := bc.Background.Padding.GetRight(10)
	dpb := bc.Background.Padding.GetBottom(50)

	return Box{
		Top:    bc.Background.Padding.GetTop(20),
		Left:   bc.Background.Padding.GetLeft(20),
		Right:  bc.GetWidth() - dpr,
		Bottom: bc.GetHeight() - dpb,
	}
}

func (bc BubbleChart) getBackgroundStyle() Style {
	return bc.Background.InheritFrom(bc.styleDefaultsBackground())
}

func (bc BubbleChart) styleDefaultsBackground() Style {
	return Style{
		FillColor:   bc.GetColorPalette().BackgroundColor(),
		StrokeColor: bc.GetColorPalette().BackgroundStrokeColor(),
		StrokeWidth: DefaultStrokeWidth,
	}
}

func (bc BubbleChart) styleDefaultsBar(index int) Style {
	return Style{
		StrokeColor: bc.GetColorPalette().GetSeriesColor(index),
		StrokeWidth: 3.0,
		FillColor:   bc.GetColorPalette().GetSeriesColor(index),
	}
}

func (bc BubbleChart) styleDefaultsTitle() Style {
	return bc.TitleStyle.InheritFrom(Style{
		FontColor:           bc.GetColorPalette().TextColor(),
		Font:                bc.GetFont(),
		FontSize:            bc.getTitleFontSize(),
		TextHorizontalAlign: TextHorizontalAlignCenter,
		TextVerticalAlign:   TextVerticalAlignTop,
		TextWrap:            TextWrapWord,
	})
}

func (bc BubbleChart) getTitleFontSize() float64 {
	effectiveDimension := util.Math.MinInt(bc.GetWidth(), bc.GetHeight())
	if effectiveDimension >= 2048 {
		return 48
	} else if effectiveDimension >= 1024 {
		return 24
	} else if effectiveDimension >= 512 {
		return 18
	} else if effectiveDimension >= 256 {
		return 12
	}
	return 10
}

func (bc BubbleChart) styleDefaultsAxes() Style {
	return Style{
		StrokeColor:         bc.GetColorPalette().AxisStrokeColor(),
		Font:                bc.GetFont(),
		FontSize:            DefaultAxisFontSize,
		FontColor:           bc.GetColorPalette().TextColor(),
		TextHorizontalAlign: TextHorizontalAlignCenter,
		TextVerticalAlign:   TextVerticalAlignTop,
		TextWrap:            TextWrapWord,
	}
}

func (bc BubbleChart) styleDefaultsElements() Style {
	return Style{
		Font: bc.GetFont(),
	}
}

// GetColorPalette returns the color palette for the chart.
func (bc BubbleChart) GetColorPalette() ColorPalette {
	if bc.ColorPalette != nil {
		return bc.ColorPalette
	}
	return AlternateColorPalette
}
