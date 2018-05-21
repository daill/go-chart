package chart

import (
	"errors"
	"io"
	"math"

	"github.com/golang/freetype/truetype"
	"github.com/wcharczuk/go-chart/util"
)

// StackedValueBar is a bar within a StackedValueBarChart.
type StackedBarValue struct {
	Name   string
	Width  int
	Values []Value
}

// GetWidth returns the width of the bar.
func (sb StackedBarValue) GetWidth() int {
	if sb.Width == 0 {
		return 50
	}
	return sb.Width
}

// StackedValueBarChart is a chart that draws sections of a bar based on percentages.
type StackedValueBarChart struct {
	Title      string
	TitleStyle Style

	ColorPalette ColorPalette

	Width  int
	Height int
	DPI    float64

	BarWidth int

	Background Style
	Canvas     Style

	XAxis Style
	YAxis YAxis

	BarSpacing int

	Font        *truetype.Font
	defaultFont *truetype.Font

	Bars     []StackedBarValue
	Elements []Renderable
}

// GetDPI returns the dpi for the chart.
func (sbc StackedValueBarChart) GetDPI(defaults ...float64) float64 {
	if sbc.DPI == 0 {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return DefaultDPI
	}
	return sbc.DPI
}

// GetFont returns the text font.
func (sbc StackedValueBarChart) GetFont() *truetype.Font {
	if sbc.Font == nil {
		return sbc.defaultFont
	}
	return sbc.Font
}

// GetWidth returns the chart width or the default value.
func (sbc StackedValueBarChart) GetWidth() int {
	if sbc.Width == 0 {
		return DefaultChartWidth
	}
	return sbc.Width
}

// GetHeight returns the chart height or the default value.
func (sbc StackedValueBarChart) GetHeight() int {
	if sbc.Height == 0 {
		return DefaultChartWidth
	}
	return sbc.Height
}

// GetBarSpacing returns the spacing between bars.
func (sbc StackedValueBarChart) GetBarSpacing() int {
	if sbc.BarSpacing == 0 {
		return 50
	}
	return sbc.BarSpacing
}

func (sbc StackedValueBarChart) drawBackground(r Renderer) {
	Draw.Box(r, Box{
		Right:  sbc.GetWidth(),
		Bottom: sbc.GetHeight(),
	}, sbc.getBackgroundStyle())
}

func (sbc StackedValueBarChart) getBackgroundStyle() Style {
	return sbc.Background.InheritFrom(sbc.styleDefaultsBackground())
}

func (sbc StackedValueBarChart) styleDefaultsBackground() Style {
	return Style{
		FillColor:   sbc.GetColorPalette().BackgroundColor(),
		StrokeColor: sbc.GetColorPalette().BackgroundStrokeColor(),
		StrokeWidth: DefaultStrokeWidth,
	}
}

// Render renders the chart with the given renderer to the given io.Writer.
func (sbc StackedValueBarChart) Render(rp RendererProvider, w io.Writer) error {
	if len(sbc.Bars) == 0 {
		return errors.New("please provide at least one bar")
	}

	r, err := rp(sbc.GetWidth(), sbc.GetHeight())
	if err != nil {
		return err
	}

	if sbc.Font == nil {
		defaultFont, err := GetDefaultFont()
		if err != nil {
			return err
		}
		sbc.defaultFont = defaultFont
	}
	r.SetDPI(sbc.GetDPI())

	sbc.drawBackground(r)

	var canvasBox Box
	var yt []Tick
	var yr Range
	var yf ValueFormatter

	canvasBox = sbc.getDefaultCanvasBox()
	yr = sbc.getRanges()
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
	yr = sbc.setRangeDomains(canvasBox, yr)
	yf = sbc.getValueFormatters()

	if sbc.hasAxes() {
		yt = sbc.getAxesTicks(r, yr, yf)
		canvasBox = sbc.getAdjustedCanvasBox(r, canvasBox, yr, yt)
		yr = sbc.setRangeDomains(canvasBox, yr)
	}

	sbc.drawCanvas(r, canvasBox)
	sbc.drawBars(r, canvasBox, yr)
	sbc.drawXAxis(r, canvasBox)
	sbc.drawYAxis(r, canvasBox, yr, yt)

	sbc.drawTitle(r)
	for _, a := range sbc.Elements {
		a(r, canvasBox, sbc.styleDefaultsElements())
	}

	return r.Save(w)
}

func (sbc StackedValueBarChart) getAxesTicks(r Renderer, yr Range, yf ValueFormatter) (yticks []Tick) {
	if sbc.YAxis.Style.Show {
		yticks = sbc.YAxis.GetTicks(r, yr, sbc.styleDefaultsAxes(), yf)
	}
	return
}


func (sbc StackedValueBarChart) hasAxes() bool {
	return sbc.YAxis.Style.Show
}


func (sbc StackedValueBarChart) getValueFormatters() ValueFormatter {
	if sbc.YAxis.ValueFormatter != nil {
		return sbc.YAxis.ValueFormatter
	}
	return FloatValueFormatter
}

func (sbc StackedValueBarChart) setRangeDomains(canvasBox Box, yr Range) Range {
	yr.SetDomain(canvasBox.Height())
	return yr
}

func (sbc StackedValueBarChart) getRanges() Range {
	var yrange Range
	if sbc.YAxis.Range != nil && !sbc.YAxis.Range.IsZero() {
		yrange = sbc.YAxis.Range
	} else {
		yrange = &ContinuousRange{}
	}

	if !yrange.IsZero() {
		return yrange
	}

	if len(sbc.YAxis.Ticks) > 0 {
		tickMin, tickMax := math.MaxFloat64, -math.MaxFloat64
		for _, t := range sbc.YAxis.Ticks {
			tickMin = math.Min(tickMin, t.Value)
			tickMax = math.Max(tickMax, t.Value)
		}
		yrange.SetMin(tickMin)
		yrange.SetMax(tickMax)
		return yrange
	}

	min, max := math.MaxFloat64, -math.MaxFloat64
	for _, b := range sbc.Bars {
		for _, c := range b.Values {
			min = math.Min(c.Value, min)
			max = math.Max(c.Value, max)
		}
	}

	yrange.SetMin(min)
	yrange.SetMax(max)

	return yrange
}

func (sbc StackedValueBarChart) drawCanvas(r Renderer, canvasBox Box) {
	Draw.Box(r, canvasBox, sbc.getCanvasStyle())
}

func (sbc StackedValueBarChart) drawBars(r Renderer, canvasBox Box, yr Range) {
	xoffset := canvasBox.Left

	width, _, _ := sbc.calculateScaledTotalWidth(canvasBox)

	var bxl, bxr int

	for _, bar := range sbc.Bars {
		barComponents := Values(bar.Values)
		bxl = xoffset
		bxr = bxl + width
		for index, bv := range barComponents {
			bxl += width
			bxr = bxl + width

			yoffset := canvasBox.Bottom - yr.Translate(bv.Value)

			barHeight := int(math.Ceil(float64(yr.Translate(bv.Value)) * float64(canvasBox.Height())))
			barBox := Box{
				Top:    yoffset,
				Left:   bxl,
				Right:  bxr,
				Bottom: util.Math.MinInt(yoffset+barHeight, canvasBox.Bottom-DefaultStrokeWidth),
			}
			Draw.Box(r, barBox, bv.Style.InheritFrom(sbc.styleDefaultsStackedValueBarValue(index)))
			yoffset += barHeight
		}

		xoffset += width + sbc.GetBarSpacing() + int((float64(len(barComponents)/2)*float64(sbc.BarWidth)))
	}
}


func (sbc StackedValueBarChart) drawXAxis(r Renderer, canvasBox Box) {
	if sbc.XAxis.Show {
		axisStyle := sbc.XAxis.InheritFrom(sbc.styleDefaultsAxes())
		axisStyle.WriteToRenderer(r)
		axisStyle.TextHorizontalAlign = TextHorizontalAlignCenter

		width, _, _ := sbc.calculateScaledTotalWidth(canvasBox)
		barComponents := sbc.getValueCountOfBars()

		r.MoveTo(canvasBox.Left, canvasBox.Bottom)
		r.LineTo(canvasBox.Right, canvasBox.Bottom)
		r.Stroke()

		r.MoveTo(canvasBox.Left, canvasBox.Bottom)
		r.LineTo(canvasBox.Left, canvasBox.Bottom+DefaultVerticalTickHeight)
		r.Stroke()

		cursor := canvasBox.Left
		for _, bar := range sbc.Bars {
			barLabelBox := Box{
				Top:    canvasBox.Bottom + DefaultXAxisMargin,
				Left:   cursor,
				Right:  (cursor + (sbc.BarWidth*barComponents))+width*2,
				Bottom: sbc.GetHeight(),
			}
			if len(bar.Name) > 0 {
				Draw.TextWithin(r, bar.Name, barLabelBox, axisStyle)
			}
			axisStyle.WriteToRenderer(r)
			r.MoveTo(barLabelBox.Right, canvasBox.Bottom)
			r.LineTo(barLabelBox.Right, canvasBox.Bottom+DefaultVerticalTickHeight)
			r.Stroke()
			cursor += width + sbc.GetBarSpacing()+int((float64(barComponents/2)*float64(sbc.BarWidth)))
		}
	}
}

func (sbc StackedValueBarChart) drawYAxis(r Renderer, canvasBox Box, yr Range, ticks []Tick) {
	if sbc.YAxis.Style.Show {
		axisStyle := sbc.YAxis.Style.InheritFrom(sbc.styleDefaultsAxes())
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
	//if sbc.YAxis.Show {
	//	axisStyle := sbc.YAxis.InheritFrom(sbc.styleDefaultsAxes())
	//	axisStyle.WriteToRenderer(r)
	//	r.MoveTo(canvasBox.Right, canvasBox.Top)
	//	r.LineTo(canvasBox.Right, canvasBox.Bottom)
	//	r.Stroke()
	//
	//	r.MoveTo(canvasBox.Right, canvasBox.Bottom)
	//	r.LineTo(canvasBox.Right+DefaultHorizontalTickWidth, canvasBox.Bottom)
	//	r.Stroke()
	//
	//	ticks := seq.RangeWithStep(0.0, 1.0, 0.2)
	//	for _, t := range ticks {
	//		axisStyle.GetStrokeOptions().WriteToRenderer(r)
	//		ty := canvasBox.Bottom - int(t*float64(canvasBox.Height()))
	//		r.MoveTo(canvasBox.Right, ty)
	//		r.LineTo(canvasBox.Right+DefaultHorizontalTickWidth, ty)
	//		r.Stroke()
	//
	//		axisStyle.GetTextOptions().WriteToRenderer(r)
	//		text := fmt.Sprintf("%0.0f%%", t*100)
	//
	//		tb := r.MeasureText(text)
	//		Draw.Text(r, text, canvasBox.Right+DefaultYAxisMargin+5, ty+(tb.Height()>>1), axisStyle)
	//	}
	//
	//}
}

func (sbc StackedValueBarChart) drawTitle(r Renderer) {
	if len(sbc.Title) > 0 && sbc.TitleStyle.Show {
		r.SetFont(sbc.TitleStyle.GetFont(sbc.GetFont()))
		r.SetFontColor(sbc.TitleStyle.GetFontColor(sbc.GetColorPalette().TextColor()))
		titleFontSize := sbc.TitleStyle.GetFontSize(DefaultTitleFontSize)
		r.SetFontSize(titleFontSize)

		textBox := r.MeasureText(sbc.Title)

		textWidth := textBox.Width()
		textHeight := textBox.Height()

		titleX := (sbc.GetWidth() >> 1) - (textWidth >> 1)
		titleY := sbc.TitleStyle.Padding.GetTop(DefaultTitleTop) + textHeight

		r.Text(sbc.Title, titleX, titleY)
	}
}

func (sbc StackedValueBarChart) getCanvasStyle() Style {
	return sbc.Canvas.InheritFrom(sbc.styleDefaultsCanvas())
}

func (sbc StackedValueBarChart) styleDefaultsCanvas() Style {
	return Style{
		FillColor:   sbc.GetColorPalette().CanvasColor(),
		StrokeColor: sbc.GetColorPalette().CanvasStrokeColor(),
		StrokeWidth: DefaultCanvasStrokeWidth,
	}
}

// GetColorPalette returns the color palette for the chart.
func (sbc StackedValueBarChart) GetColorPalette() ColorPalette {
	if sbc.ColorPalette != nil {
		return sbc.ColorPalette
	}
	return AlternateColorPalette
}

func (sbc StackedValueBarChart) getDefaultCanvasBox() Box {
	return sbc.box()
}

func (sbc StackedValueBarChart) calculateScaledTotalWidth(canvasBox Box) (width, spacing, total int) {
	spacing = sbc.calculateEffectiveBarSpacing(canvasBox)
	width = sbc.calculateEffectiveBarWidth(canvasBox, spacing)
	total = sbc.calculateTotalBarWidth(width, spacing)
	return
}

func (sbc StackedValueBarChart) GetBarWidth() int {
	if sbc.BarWidth == 0 {
		return DefaultBarWidth
	}
	return sbc.BarWidth
}

func (sbc StackedValueBarChart) calculateEffectiveBarSpacing(canvasBox Box) int {
	totalWithBaseSpacing := sbc.calculateTotalBarWidth(sbc.GetBarWidth(), sbc.GetBarSpacing())
	if totalWithBaseSpacing > canvasBox.Width() {
		lessBarWidths := canvasBox.Width() - ((len(sbc.Bars) * (sbc.getValueCountOfBars() * sbc.GetBarWidth())))
		if lessBarWidths > 0 {
			return int(math.Ceil(float64(lessBarWidths) / float64(len(sbc.Bars) *sbc.getValueCountOfBars())))
		}
		return 0
	}
	return sbc.GetBarSpacing()
}

func (sbc StackedValueBarChart) calculateEffectiveBarWidth(canvasBox Box, spacing int) int {
	totalWithBaseWidth := sbc.calculateTotalBarWidth(sbc.GetBarWidth(), spacing)
	if totalWithBaseWidth > canvasBox.Width() {
		totalLessBarSpacings := canvasBox.Width() - (len(sbc.Bars) + spacing)
		if totalLessBarSpacings > 0 {
			return int(math.Ceil(float64(totalLessBarSpacings) / float64(len(sbc.Bars)*sbc.getValueCountOfBars())))
		}
		return 0
	}
	return sbc.GetBarWidth()
}

func (sbc StackedValueBarChart) getValueCountOfBars() int {
	maxCount := -math.MaxFloat64
	for _, bar := range sbc.Bars {
		maxCount = math.Max(float64(maxCount), float64(len(bar.Values)))
	}
	return int(maxCount)
}

func (sbc StackedValueBarChart) calculateTotalBarWidth(barWidth, spacing int) int {
	return ((barWidth*sbc.getValueCountOfBars()) + spacing)*len(sbc.Bars)
}

func (sbc StackedValueBarChart) getAdjustedCanvasBox(r Renderer, canvasBox Box, yrange Range, yticks []Tick) Box {
	axesOuterBox := canvasBox.Clone()

	_, _, totalWidth := sbc.calculateScaledTotalWidth(canvasBox)

	if sbc.XAxis.Show {
		xaxisHeight := DefaultVerticalTickHeight

		axisStyle := sbc.XAxis.InheritFrom(sbc.styleDefaultsAxes())
		axisStyle.WriteToRenderer(r)

		cursor := canvasBox.Left
		for _, bar := range sbc.Bars {
			if len(bar.Name) > 0 {
				barLabelBox := Box{
					Top:    canvasBox.Bottom + DefaultXAxisMargin,
					Left:   cursor,
					Right:  cursor + (bar.GetWidth()*len(bar.Values)) + sbc.GetBarSpacing(),
					Bottom: sbc.GetHeight(),
				}
				lines := Text.WrapFit(r, bar.Name, barLabelBox.Width(), axisStyle)
				linesBox := Text.MeasureLines(r, lines, axisStyle)

				xaxisHeight = util.Math.MaxInt(linesBox.Height()+(2*DefaultXAxisMargin), xaxisHeight)
			}
		}
		xbox := Box{
			Top:    canvasBox.Top,
			Left:   canvasBox.Left,
			Right:  canvasBox.Left + totalWidth,
			Bottom: sbc.GetHeight() - xaxisHeight,
		}
		axesOuterBox = axesOuterBox.Grow(xbox)
	}


	if sbc.YAxis.Style.Show {
		axesBounds := sbc.YAxis.Measure(r, canvasBox, yrange, sbc.styleDefaultsAxes(), yticks)
		axesOuterBox = axesOuterBox.Grow(axesBounds)
	}

	return canvasBox.OuterConstrain(sbc.box(), axesOuterBox)

}

func (sbc StackedValueBarChart) box() Box {
	dpr := sbc.Background.Padding.GetRight(10)
	dpb := sbc.Background.Padding.GetBottom(50)

	return Box{
		Top:    sbc.Background.Padding.GetTop(20),
		Left:   sbc.Background.Padding.GetLeft(20),
		Right:  sbc.GetWidth() - dpr,
		Bottom: sbc.GetHeight() - dpb,
	}
}

func (sbc StackedValueBarChart) styleDefaultsStackedValueBarValue(index int) Style {
	return Style{
		StrokeColor: sbc.GetColorPalette().GetSeriesColor(index),
		StrokeWidth: 3.0,
		FillColor:   sbc.GetColorPalette().GetSeriesColor(index),
	}
}

func (sbc StackedValueBarChart) styleDefaultsTitle() Style {
	return sbc.TitleStyle.InheritFrom(Style{
		FontColor:           sbc.GetColorPalette().TextColor(),
		Font:                sbc.GetFont(),
		FontSize:            sbc.getTitleFontSize(),
		TextHorizontalAlign: TextHorizontalAlignCenter,
		TextVerticalAlign:   TextVerticalAlignTop,
		TextWrap:            TextWrapWord,
	})
}

func (sbc StackedValueBarChart) getTitleFontSize() float64 {
	effectiveDimension := util.Math.MinInt(sbc.GetWidth(), sbc.GetHeight())
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

func (sbc StackedValueBarChart) styleDefaultsAxes() Style {
	return Style{
		StrokeColor:         sbc.GetColorPalette().AxisStrokeColor(),
		Font:                sbc.GetFont(),
		FontSize:            DefaultAxisFontSize,
		FontColor:           sbc.GetColorPalette().TextColor(),
		TextHorizontalAlign: TextHorizontalAlignCenter,
		TextVerticalAlign:   TextVerticalAlignTop,
		TextWrap:            TextWrapWord,
	}
}
func (sbc StackedValueBarChart) styleDefaultsElements() Style {
	return Style{
		Font: sbc.GetFont(),
	}
}
