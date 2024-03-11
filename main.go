// Create this: https://app-h5.govee.com/user-manual/wlan-guide

// Initialise Go program
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	log "github.com/charmbracelet/log"
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
		Cmd string `json:"cmd"`
		// Add other info for bubbletea model
		DeviceData struct {
			Device string       `json:"device"`
			IP     *net.UDPAddr `json:"ip"`
			Sku    string       `json:"sku"`
		} `json:"deviceData"`
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

var UdpServerAddr *net.UDPAddr
var UdpScanAddr *net.UDPAddr

func main() {
	// Populate the global UDP addresses
	serverAddr := "239.255.255.250:4002"
	scanAddr := "239.255.255.250:4001"

	// Resolve the string server address to a UDP address
	udpServerAddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	UdpServerAddr = udpServerAddr

	// Resolve the string scan address to a UDP address
	udpScanAddr, err := net.ResolveUDPAddr("udp", scanAddr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	UdpScanAddr = udpScanAddr

	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

// Testing out bubbletea
// Model should create a list of govee devices and allow the user to select one or more
// Start with just toggling state of selected devices

type model struct {
	choices []DevStatus // lamps that have reported
	cursor  int         // which lamp our cursor is pointing at
}

func initialModel() model {
	// Start listening for UDP packages on the server address
	conn, err := net.ListenUDP("udp", UdpServerAddr)
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
		os.Exit(1)
	}

	// Send the JSON data over UDP
	log.Info("Sending UDP data to", UdpScanAddr)
	_, err = conn.WriteToUDP(jsonData, UdpScanAddr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Read from UDP listener
	// TODO Read all devices not just the first to respond(?)
	var buf [512]byte
	n, _, err := conn.ReadFromUDP(buf[0:])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Unmarshal the data into a ScanResponse struct
	var scanResponseData ScanResponse
	err = json.Unmarshal(buf[0:n], &scanResponseData)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Create a SimpleStateUpdateRequest devStatus instance
	devStatusUpdateData := SimpleStateUpdateRequest{
		Msg: struct {
			Cmd  string `json:"cmd"`
			Data struct {
				Value int `json:"value"`
			} `json:"data"`
		}{
			Cmd: "devStatus",
		},
	}

	// Marshal the data into JSON
	devStatusRequestData, err := json.Marshal(devStatusUpdateData)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Resolve the device address to a UDP address
	deviceAddr, err := net.ResolveUDPAddr("udp", scanResponseData.Msg.Data.IP+":4003")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Send the JSON data over UDP
	log.Info("Sending devStatus request to", deviceAddr)
	_, err = conn.WriteToUDP(devStatusRequestData, deviceAddr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	//read from UDP listener
	var devStatusBuf [512]byte
	n, _, err = conn.ReadFromUDP(devStatusBuf[0:])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Unmarshal the data into a DevStatus struct
	var devStatusResponseData DevStatus
	err = json.Unmarshal(devStatusBuf[0:n], &devStatusResponseData)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Expand the data object with the device, IP and SKU
	devStatusResponseData.Msg.DeviceData.Device = scanResponseData.Msg.Data.Device
	devStatusResponseData.Msg.DeviceData.IP = deviceAddr
	devStatusResponseData.Msg.DeviceData.Sku = scanResponseData.Msg.Data.Sku

	// Pretty print the data
	prettyData, err := json.MarshalIndent(devStatusResponseData, "", "  ")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(string(prettyData))

	return model{
		choices: []DevStatus{devStatusResponseData},
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
			err := m.choices[m.cursor].ToggleDeviceState()
			if err != nil {
				// TODO: Handle error on TUI
				fmt.Println(err)
			}

		}
	}

	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, nil
}

func (m model) View() string {
	// The header
	s := "The following devices responded to the scan request:\n\n"

	// Iterate over our choices
	for i, choice := range m.choices {

		// Is the cursor pointing at this choice?
		cursor := " " // no cursor
		if m.cursor == i {
			cursor = ">" // cursor!
		}

		// Is this choice selected?
		checked := " " // not selected
		if choice.Msg.Data.OnOff == 1 {
			checked = "x" // selected!
		}

		// Render the row
		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice.Msg.DeviceData.IP)
	}

	// The footer
	s += "\nPress q to quit.\n"

	// Send the UI for rendering
	return s
}

func (m *DevStatus) ToggleDeviceState() error {
	// Start listening for UDP packages on the server address
	conn, err := net.ListenUDP("udp", UdpServerAddr)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// Create an instance of SimpleStateUpdateRequest
	toggleStateData := SimpleStateUpdateRequest{
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
				Value: int(math.Abs(float64(m.Msg.Data.OnOff - 1))),
			},
		},
	}

	// Update the device state
	m.Msg.Data.OnOff = toggleStateData.Msg.Data.Value

	// Marshal the data into JSON
	toggleStateJson, err := json.Marshal(toggleStateData)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// Send the JSON data over UDP
	_, err = conn.WriteToUDP(toggleStateJson, m.Msg.DeviceData.IP)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}
