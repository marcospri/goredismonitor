package main

import ui "gopkg.in/gizak/termui.v1"

const (
	defaultTheme = "helloworld"
)

func CreateUI(theme string) *Gui {
	err := ui.Init()
	if err != nil {
		panic(err)
	}

	ui.UseTheme(theme)

	bc := ui.NewBarChart()
	bc.Width = 50
	bc.Height = 10
	bc.BarWidth = 6
	bc.Border.Label = "Top commands"

	sl := ui.NewSparkline()
	sl.Height = 7
	sll := ui.NewSparklines(sl)
	sll.Width = 50
	sll.Height = 10
	sll.Border.Label = "Command rate"

	ll := ui.NewList()
	ll.Width = 100
	ll.Height = 15
	ll.Border.Label = "Firehose"

	st := ui.NewPar("")
	st.Border.Label = "Status"
	st.Height = 3
	st.Width = 100

	ui.Body.AddRows(
		ui.NewRow(
			ui.NewCol(6, 0, bc),
			ui.NewCol(6, 0, sll),
		),
		ui.NewRow(
			ui.NewCol(12, 0, ll),
		),
		ui.NewRow(
			ui.NewCol(12, 0, st),
		),
	)
	ui.Body.Align()

	return &Gui{
		fireHose:  ll,
		statusBar: st,
		rateGraph: sll,
		cmdsGraph: bc,
	}
}

func CloseUI() {
	ui.Close()
}

type Gui struct {
	fireHose  *ui.List
	statusBar *ui.Par
	cmdsGraph *ui.BarChart
	rateGraph *ui.Sparklines
}

func (g *Gui) setStatus(status string) {
	g.statusBar.Text = status
}

func (g *Gui) updateCmdCountsGraph(data []int, labels []string) {
	g.cmdsGraph.Data = data
	g.cmdsGraph.DataLabels = labels
}

func (g *Gui) updateFirehose(lines []string) {
	g.fireHose.Items = lines
}

func (g *Gui) updateRateGraph(data []int) {
	g.rateGraph.Lines[0].Data = data
}

func (g *Gui) render() {
	ui.Render(ui.Body)
}
