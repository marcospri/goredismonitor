package main

//TODO

// Flag for redis host
// Auto adjust number of cmds

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	ui "gopkg.in/gizak/termui.v1"
)

func GetTopCmd(cmdStats map[string]int, topN int) ([]int, []string) {
	data := make([]int, 0, len(cmdStats))
	labels := make([]string, 0, len(cmdStats))

	items := make([]CmdFrequency, 0, len(cmdStats))

	for k, v := range cmdStats {
		items = append(items, CmdFrequency{k, v})
	}

	sort.Sort(SortedFrequency(items))

	for _, v := range items {
		data = append(data, v.Frequency)
		labels = append(labels, v.Cmd)
	}
	return data, labels
}

func (a SortedFrequency) Len() int           { return len(a) }
func (a SortedFrequency) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SortedFrequency) Less(i, j int) bool { return a[i].Frequency > a[j].Frequency }

type SortedFrequency []CmdFrequency

type CmdFrequency struct {
	Cmd       string
	Frequency int
}

type CmdInfo struct {
	Cmd    string
	Params []string
	When   time.Time
	RawCmd string
}

func (c *CmdInfo) String() string {
	return fmt.Sprintf("[%s] - %s %s", c.When.Format(time.RFC3339Nano), c.Cmd, strings.Join(c.Params, " "))
}

var monitorR = `\+(\d{10})\.\d{6} \[0 (\d{0,3}\.\d{0,3}\.\d{0,3}\.\d{0,3}):(\d{5})\] ((\"((.*)?)\") ?)+`
var monitorRegex = regexp.MustCompile(monitorR)

func parseMonitorLine(monitorLine string) *CmdInfo {
	info := CmdInfo{}

	matches := monitorRegex.FindAllStringSubmatch(monitorLine, -1)
	if matches != nil && len(matches[0]) == 8 {
		info.RawCmd = monitorLine
		cmdParts := strings.Split(matches[0][5], " ")

		info.Cmd = cmdParts[0]
		info.Params = cmdParts[1:]

		timestamp, err := strconv.ParseInt(matches[0][1], 10, 64)
		if err != nil {
			return nil
		}
		info.When = time.Unix(timestamp, 0)
	} else {
		return nil
	}
	return &info
}

func countEvents(cmdTimeStats []int, lastTimeStats int) []int {
	return append(cmdTimeStats, lastTimeStats)
}

func addFirehoseCmd(cmdList []string, cmdInfo *CmdInfo) []string {
	if len(cmdList) == 100 {
		cmdList = cmdList[10:]
	}
	cmdList = append(cmdList, cmdInfo.String())

	return cmdList
}

func cmdListener(scanner *bufio.Scanner, cmds chan *CmdInfo) {
	for scanner.Scan() {
		info := parseMonitorLine(scanner.Text())
		cmds <- info
	}
	close(cmds)
}

func main() {
	flagTheme := flag.String("theme", defaultTheme,
		"UI theme to use, options are helloworld or default.")
	uiRefreshRate := flag.Int64("uirefresh", 500, "UI refresh rate, in miliseconds.")
	statsResolution := flag.Int64("statsresolution", 500, "Resolution for the For the cmd/time graph in miliseconds")
	redisPort := flag.Int("port", 6379, "Redis port.")

	flag.Parse()

	if *flagTheme != "helloworld" && *flagTheme != "default" {
		*flagTheme = defaultTheme
	}

	//Connect to redis
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", *redisPort))
	if err != nil {
		panic(err)
	}
	gui := CreateUI(*flagTheme)
	defer CloseUI()

	statusRunning := fmt.Sprintf("Monitoring redis at %d", *redisPort)
	statusPaused := "[PAUSED]" + statusRunning
	gui.setStatus(statusRunning)

	//Render empty UI
	gui.render()

	var currentEvents int
	var paused bool
	allTimeStats := make([]int, 0, 1000)
	cmdNameStats := make(map[string]int)
	cmdList := make([]string, 0, 100)

	uiRefreshTicker := time.Tick(time.Duration(*uiRefreshRate) * time.Millisecond)
	statsTicker := time.Tick(time.Duration(*statsResolution) * time.Millisecond)

	//Start the actual redis monitoring
	fmt.Fprintf(conn, "MONITOR\r\n")
	scanner := bufio.NewScanner(conn)
	cmdChan := make(chan *CmdInfo, 1000)
	go cmdListener(scanner, cmdChan)

	evt := ui.EventCh()

	for {
		select {
		//Recive new commands from redis monitor
		case cmdInfo := <-cmdChan:
			if cmdInfo != nil {
				cmdList = addFirehoseCmd(cmdList, cmdInfo)

				currentEvents++
				cmdNameStats[cmdInfo.Cmd]++

				//Update graph data
				graphData, graphLabels := GetTopCmd(cmdNameStats, 5)
				gui.updateCmdCountsGraph(graphData, graphLabels)

				gui.updateFirehose(cmdList)
				gui.updateRateGraph(allTimeStats)
			}
		//Recieve UI events
		case e := <-evt:
			//Quit
			if e.Type == ui.EventKey && e.Ch == 'q' {
				return
			}
			//Pause
			if e.Type == ui.EventKey && e.Ch == 'p' {
				paused = !paused
				if paused {
					gui.setStatus(statusPaused)
				} else {
					gui.setStatus(statusRunning)
				}
				ui.Render(ui.Body)
			}

			if e.Type == ui.EventResize {
				ui.Body.Width = ui.TermWidth()
				ui.Body.Align()
			}
		//Calculate cmds rate
		case <-statsTicker:
			allTimeStats = countEvents(allTimeStats, currentEvents)
			currentEvents = 0

			gui.updateRateGraph(allTimeStats)

		//Refresh UI
		case <-uiRefreshTicker:
			if !paused {
				ui.Render(ui.Body)
			}
		}
	}
}
