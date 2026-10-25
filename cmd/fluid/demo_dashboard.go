package main

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/render"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/terminal"
	"github.com/unaquatic/fluid/widget"
)

var demoDashboardCmd = Demo{
	Name:  "dashboard",
	Short: "Multi-panel dashboard demo",
	Run:   runDashboardDemo,
}

func runDashboardDemo(ctx context.Context, o DemoOptions) error {
	r, err := terminal.NewRenderer(terminal.Config{
		UseMouse: !o.NoMouse,
	})
	if err != nil {
		return fmt.Errorf("renderer: %w", err)
	}

	root := r.Root()

	tabs := widget.NewTabSelect([]widget.TabSelectItem{
		{Name: "Overview"},
		{Name: "Metrics"},
		{Name: "Logs"},
		{Name: "Settings"},
	})

	sidebarList := widget.NewList([]widget.ListItem{
		{Text: "System"},
		{Text: "Network"},
		{Text: "Storage"},
		{Text: "Processes"},
		{Text: "Services"},
		{Text: "Security"},
	}, widget.ListWithLayout(layout.WithFlexGrow(1)))

	type metricGauge struct {
		label string
		gauge *widget.Gauge
		base  float64
	}
	gauges := []metricGauge{
		{label: "CPU", base: 0.65},
		{label: "MEM", base: 0.52},
		{label: "DSK", base: 0.31},
		{label: "NET", base: 0.44},
	}

	gaugeLines := make([]render.Renderable, len(gauges))
	for i := range gauges {
		g := widget.NewGauge(
			widget.GaugeWithValue(gauges[i].base),
			widget.GaugeWithLayout(layout.WithFlexGrow(1)),
		)
		gauges[i].gauge = g

		gaugeLines[i] = widget.NewBox(
			widget.BoxWithLayout(
				layout.WithFlexDirection(layout.DirectionRow),
				layout.WithHeight(layout.Points(1)),
				layout.WithAlignItems(layout.AlignCenter),
				layout.WithGap(1),
			),
			widget.Children(
				widget.NewText(
					fmt.Sprintf("%-4s", gauges[i].label),
					widget.TextWithLayout(layout.WithWidth(layout.Points(5))),
				),
				g,
			),
		)
	}

	initialData := make([]float64, 40)
	for i := range initialData {
		initialData[i] = 0.5 + 0.3*math.Sin(float64(i)*0.3)
	}
	sparkline := widget.NewSparkline(initialData,
		widget.SparklineWithMaxValue(1.0),
		widget.SparklineWithLayout(layout.WithFlexGrow(1)),
	)

	eventsTable := widget.NewTable(
		[]widget.TableColumn{
			{Name: "Time", Width: 8},
			{Name: "Event", Width: 0},
			{Name: "Status", Width: 8},
		},
		[][]string{
			{"14:23", "Service started", "OK"},
			{"14:21", "Config reloaded", "OK"},
			{"14:19", "Connection established", "OK"},
			{"14:15", "Cache cleared", "OK"},
			{"14:12", "Auth token refreshed", "OK"},
			{"14:10", "Health check passed", "OK"},
			{"14:05", "Backup completed", "OK"},
			{"14:00", "Scheduler tick", "OK"},
			{"13:55", "Log rotated", "OK"},
			{"13:50", "Metrics exported", "OK"},
		},
		widget.TableWithStripedRows(true),
		widget.TableWithLayout(layout.WithFlexGrow(1)),
	)

	footerUptime := widget.NewText("")

	sidebarBox := widget.NewBox(
		widget.BoxWithLayout(
			layout.WithWidth(layout.Points(20)),
			layout.WithFlexDirection(layout.DirectionColumn),
			layout.WithPadding(layout.Points(1)),
			layout.WithGap(1),
		),
		widget.BoxWithBorder(style.BorderSides(style.SideRight)),
		widget.BoxWithBorderStyle(style.BorderSingle),
		widget.Children(
			widget.NewText("Navigation", widget.TextWithLayout(layout.WithHeight(layout.Points(1)))),
			sidebarList,
		),
	)

	sparkPanel := widget.NewBox(
		widget.BoxWithLayout(
			layout.WithFlexGrow(1),
			layout.WithFlexDirection(layout.DirectionColumn),
			layout.WithPadding(layout.Points(1)),
		),
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderSingle),
		widget.BoxWithTitle("CPU History"),
		widget.Children(sparkline),
	)

	tablePanel := widget.NewBox(
		widget.BoxWithLayout(
			layout.WithFlexGrow(1),
			layout.WithFlexDirection(layout.DirectionColumn),
		),
		widget.BoxWithBorder(style.BorderAll()),
		widget.BoxWithBorderStyle(style.BorderSingle),
		widget.BoxWithTitle("Recent Events"),
		widget.Children(eventsTable),
	)

	gaugeRowChildren := []render.Renderable{
		widget.NewText("System Metrics", widget.TextWithLayout(layout.WithHeight(layout.Points(1)))),
	}
	gaugeRowChildren = append(gaugeRowChildren, gaugeLines...)

	outer := widget.NewBox(
		widget.BoxWithLayout(
			layout.WithFlexGrow(1),
			layout.WithFlexDirection(layout.DirectionColumn),
		),
		widget.Children(
			widget.NewBox(
				widget.BoxWithLayout(
					layout.WithHeight(layout.Points(3)),
					layout.WithFlexDirection(layout.DirectionRow),
					layout.WithAlignItems(layout.AlignCenter),
					layout.WithPadding(layout.Points(1)),
					layout.WithGap(2),
				),
				widget.BoxWithBorder(style.BorderSides(style.SideBottom)),
				widget.BoxWithBorderStyle(style.BorderSingle),
				widget.Children(
					widget.NewText("Dashboard", widget.TextWithLayout(layout.WithWidth(layout.Points(12)))),
					tabs,
				),
			),

			widget.NewBox(
				widget.BoxWithLayout(
					layout.WithFlexGrow(1),
					layout.WithFlexDirection(layout.DirectionRow),
				),
				widget.Children(
					sidebarBox,
					widget.NewBox(
						widget.BoxWithLayout(
							layout.WithFlexGrow(1),
							layout.WithFlexDirection(layout.DirectionColumn),
							layout.WithPadding(layout.Points(1)),
							layout.WithGap(1),
						),
						widget.Children(
							widget.NewBox(
								widget.BoxWithLayout(
									layout.WithFlexDirection(layout.DirectionColumn),
									layout.WithGap(1),
								),
								widget.Children(gaugeRowChildren...),
							),

							widget.NewBox(
								widget.BoxWithLayout(
									layout.WithFlexGrow(1),
									layout.WithFlexDirection(layout.DirectionRow),
									layout.WithGap(2),
								),
								widget.Children(sparkPanel, tablePanel),
							),
						),
					),
				),
			),

			widget.NewBox(
				widget.BoxWithLayout(
					layout.WithHeight(layout.Points(1)),
					layout.WithFlexDirection(layout.DirectionRow),
					layout.WithJustifyContent(layout.JustifySpaceBetween),
				),
				widget.Children(
					widget.NewText(" Connected"),
					footerUptime,
					widget.NewText("v1.0.0 "),
				),
			),
		),
	)
	root.Add(outer)

	focusables := []render.Renderable{tabs, sidebarList, eventsTable}
	focusIdx := 0

	r.FocusRenderable(focusables[focusIdx])

	startTime := time.Now()

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		tick := 0
		for range ticker.C {
			if r.IsDestroyed() {
				return
			}
			tick++

			for i := range gauges {
				jitter := (rand.Float64() - 0.5) * 0.08
				v := gauges[i].base + jitter + 0.05*math.Sin(float64(tick)*0.2+float64(i))
				if v < 0 {
					v = 0
				}
				if v > 1 {
					v = 1
				}
				gauges[i].gauge.SetValue(v)
			}

			cpuVal := 0.5 + 0.3*math.Sin(float64(tick)*0.15) + (rand.Float64()-0.5)*0.1
			if cpuVal < 0 {
				cpuVal = 0
			}
			if cpuVal > 1 {
				cpuVal = 1
			}
			sparkline.AddPoint(cpuVal)

			elapsed := time.Since(startTime)
			hours := int(elapsed.Hours())
			mins := int(elapsed.Minutes()) % 60
			secs := int(elapsed.Seconds()) % 60
			footerUptime.SetContent(fmt.Sprintf("Uptime: %dh %dm %ds", hours, mins, secs))

			r.RequestRender()
		}
	}()

	r.EventBus().On(input.EventKeyboard, func(e input.TerminalEvent) {
		if e.Key == nil {
			return
		}
		k := e.Key

		if k.Name == "escape" || (k.Name == "q" && !k.Mods.Has(input.ModCtrl) && !k.Mods.Has(input.ModAlt)) {
			r.Destroy()
			return
		}
		if k.Name == "tab" && !k.Mods.Has(input.ModCtrl) && !k.Mods.Has(input.ModAlt) {
			k.PreventDefault()
			focusIdx = (focusIdx + 1) % len(focusables)
			r.FocusRenderable(focusables[focusIdx])
			return
		}
	})

	return r.RunContinuous(ctx)
}
