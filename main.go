// Create this: https://app-h5.govee.com/user-manual/wlan-guide

// Initialise Go program
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

type ScanRequest struct {
	Msg struct {
		Cmd  string `json:"cmd"`
		Data struct {
			AccountTopic string `json:"account_topic"`
		} `json:"data"`
	} `json:"msg"`
}

type SimpleStateUpdateRequest struct {
	Msg struct {
		Cmd  string `json:"cmd"`
		Data struct {
			Value int `json:"value"`
		} `json:"data"`
	} `json:"msg"`
}

type ScanResponse struct {
	Msg struct {
		Cmd  string `json:"cmd"`
		Data struct {
			IP              string `json:"ip"`
			Device          string `json:"device"`
			Sku             string `json:"sku"`
			BleVersionHard  string `json:"bleVersionHard"`
			BleVersionSoft  string `json:"bleVersionSoft"`
			WifiVersionHard string `json:"wifiVersionHard"`
			WifiVersionSoft string `json:"wifiVersionSoft"`
		} `json:"data"`
	} `json:"msg"`
}

type DevStatus struct {
	Msg struct {
		Cmd  string `json:"cmd"`
		Data struct {
			OnOff      int `json:"onOff"`
			Brightness int `json:"brightness"`
			Color      struct {
				R int `json:"r"`
				G int `json:"g"`
				B int `json:"b"`
			} `json:"color"`
			ColorTemInKelvin int `json:"colorTemInKelvin"`
		} `json:"data"`
	} `json:"msg"`
}

func main() {

	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}

	serverAddr := "239.255.255.250:4002"
	scanAddr := "239.255.255.250:4001"

	// Resolve the string address to a UDP address
	udpServerAddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Resolve the string address to a UDP address
	udpScanAddr, err := net.ResolveUDPAddr("udp", scanAddr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Start listening for UDP packages on the given address
	conn, err := net.ListenUDP("udp", udpServerAddr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Create an instance of ScanRequest
	data := ScanRequest{
		Msg: struct {
			Cmd  string `json:"cmd"`
			Data struct {
				AccountTopic string `json:"account_topic"`
			} `json:"data"`
		}{
			Cmd: "scan",
			Data: struct {
				AccountTopic string `json:"account_topic"`
			}{
				AccountTopic: "reserve",
			},
		},
	}

	// Marshal the data into JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Send the JSON data over UDP
	log.Println("Sending UDP data to", udpScanAddr)
	_, err = conn.WriteToUDP(jsonData, udpScanAddr)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Read from UDP listener
	var buf [512]byte
	n, _, err := conn.ReadFromUDP(buf[0:])
	if err != nil {
		fmt.Println(err)
		return
	}

	// Unmarshal the data into a ScanResponse struct
	var responseData ScanResponse
	err = json.Unmarshal(buf[0:n], &responseData)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Pretty print the data
	prettyData, err := json.MarshalIndent(responseData, "", "  ")
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(prettyData))

	var deviceAddr = responseData.Msg.Data.IP + ":4003"

	// Resolve the string address to a UDP address
	udpDeviceAddr, err := net.ResolveUDPAddr("udp", deviceAddr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Create an instance of SimpleStateUpdateRequest
	updateData := SimpleStateUpdateRequest{
		Msg: struct {
			Cmd  string `json:"cmd"`
			Data struct {
				Value int `json:"value"`
			} `json:"data"`
		}{
			Cmd: "turn",
			Data: struct {
				Value int `json:"value"`
			}{
				Value: 0,
			},
		},
	}

	// Marshal the data into JSON
	requestData, err := json.Marshal(updateData)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Send the JSON data over UDP
	log.Println("Sending UDP data to", udpDeviceAddr)
	_, err = conn.WriteToUDP(requestData, udpDeviceAddr)
	if err != nil {
		fmt.Println(err)
		return
	}

}

// Testing out bubbletea
// Model should create a list of govee devices and allow the user to select one or more
// Start with just toggling state of selected devices

type model struct {
	choices  []string         // items on the to-do list
	cursor   int              // which to-do list item our cursor is pointing at
	selected map[int]struct{} // which to-do items are selected
}

func initialModel() model {
	return model{
		// Our to-do list is a grocery list
		choices: []string{"Buy carrots", "Buy celery", "Buy kohlrabi"},

		// A map which indicates which choices are selected. We're using
		// the map like a mathematical set. The keys refer to the indexes
		// of the `choices` slice, above.
		selected: make(map[int]struct{}),
	}
}

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Is it a key press?
	case tea.KeyMsg:

		// Cool, what was the actual key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit

		// The "up" and "k" keys move the cursor up
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		// The "down" and "j" keys move the cursor down
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		// The "enter" key and the spacebar (a literal space) toggle
		// the selected state for the item that the cursor is pointing at.
		case "enter", " ":
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		}
	}

	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, nil
}

func (m model) View() string {
	// The header
	s := "What should we buy at the market?\n\n"

	// Iterate over our choices
	for i, choice := range m.choices {

		// Is the cursor pointing at this choice?
		cursor := " " // no cursor
		if m.cursor == i {
			cursor = ">" // cursor!
		}

		// Is this choice selected?
		checked := " " // not selected
		if _, ok := m.selected[i]; ok {
			checked = "x" // selected!
		}

		// Render the row
		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
	}

	// The footer
	s += "\nPress q to quit.\n"

	// Send the UI for rendering
	return s
}
