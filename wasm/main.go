// GOOS=js GOARCH=wasm go build -o  ../assets/hive.wasm

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"syscall/js"
	"time"

	"github.com/hako/durafmt"
	logger "github.com/ipfs/go-log/v2"
)

var log = logger.Logger("hive-wasm")
var StartTime int64
const (
	EVENTS  = "http://localhost:4343/v3/events"
	GATEWAY = "http://localhost:4343/v3/execute"
	splicer = "%$#"
)

func Events() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			log.Debug("Events Called")
			resp, err := http.Post(EVENTS, "application/json", nil)
			if err != nil {
				log.Error(err.Error())
				return
			}
			defer resp.Body.Close()
			reader := bufio.NewReader(resp.Body)
			for {
				var eventsDataString string
				line, isPrefix, err := reader.ReadLine()
				log.Debugf("This is line: %s", string(line))
				if string(line) == "" {
					log.Debug("Empty Response at reader.ReadLine")
					return
				}
				if err != nil {
					log.Debugf("Error in reading data string: %s", err.Error())
					continue
				}
				eventsDataString = string(line)
				for isPrefix {
					line, isPrefix, err = reader.ReadLine()
					if err != nil {
						log.Debugf("Error in reading prefixed data string: %s", err.Error())
						continue
					}
					eventsDataString += string(line)
				}
				log.Debugf("This is the Events Data String: %+v", eventsDataString)
				var event Event
				err = json.Unmarshal([]byte(eventsDataString), &event)
				if err != nil {
					log.Error("Error in Unmarshalling eventsDataString:", err.Error())
					return
				}
				var out Out
				log.Debugf("This is event: %s", event.Result.Topic)
				err = json.Unmarshal([]byte(event.Result.Val), &out)
				if err != nil {
					log.Error("Error in Unmarshalling Out in : ", event.Result.Val, err.Error())
					return
				}
				val, err := json.Marshal(out.Data)
				if err != nil {
					log.Error("Error encountered in Marshalling: ", err.Error())
					return
				}
				switch event.Result.Topic {
				case "Status":
					{
						log.Debug("Status Hit")
						var status Status
						err = json.Unmarshal(val, &status)
						if err != nil {
							log.Error("Error in Unmarshalling Status:", err.Error())
							return
						}
						log.Debug("This is Status: ", status)
						jsDoc := js.Global().Get("document")
						if !jsDoc.Truthy() {
							log.Error("Unable to get document object in status")
							return
						}
						SetDisplay("taskmanagerstatusname", "innerHTML", "")
						SetDisplay("taskmanagerstatusstatus", "innerHTML", "")
						SetDisplay("taskmanagerstatusAS", "innerHTML", "")
						for _, task := range status.TaskManagerStatus {
							sName := task.Name
							if sName == "Idle" {
								continue
							}
							CreateElement("taskmanagerstatusname", "div", "innerHTML", sName)
							sStatus := task.Status
							CreateElement("taskmanagerstatusstatus", "div", "innerHTML", sStatus)
							sAdditionalStatus := task.AdditionalStatus
							if sAdditionalStatus == "" {
								sAdditionalStatus = fmt.Sprintf("&#8212;")
							}
							CreateElement("taskmanagerstatusAS", "div", "innerHTML", sAdditionalStatus)
						}
						serverStatus := reflect.ValueOf(&status.ServerDetails).Elem()
						for key := 0; key < serverStatus.NumField(); key++ {
							name := serverStatus.Type().Field(key).Name
							value := serverStatus.Field(key).Interface()
							if value == "" {
								value = "Not Running"
							}
							SetDisplay(name, "innerHTML", fmt.Sprintf("%s", value))
						}
						values := reflect.ValueOf(&status).Elem()
						for key := 0; key < values.NumField(); key++ {
							name := values.Type().Field(key).Name
							value := values.Field(key).Interface()
							if (name == "TaskManagerStatus") || (name == "TotalUptimePercentage") || (name == "SessionStartTime") || (name == "ServerDetails") {
								continue
							}
							var sValue string
							if value == true {
								switch name {
								case "LoggedIn":
									sValue = "LoggedIn"
								case "DaemonRunning":
									sValue = "ONLINE"
								}
							} else if value == false {
								switch name {
								case "LoggedIn":
									sValue = "LoggedOut"
								case "DaemonRunning":
									sValue = "OFFLINE"
								}
							}
							SetDisplay(name, "innerHTML", sValue)
						}
						sFloat := fmt.Sprintf("%.2f", status.TotalUptimePercentage.Percentage)
						sValue := fmt.Sprintf("%s %s", sFloat, "%")
						SetDisplay("percentageNumber", "innerHTML", sValue)
						StartTime = status.SessionStartTime
						log.Debug("Daemon Started at: ", StartTime)
						CheckBanner()
					}
				case "Balance":
					{
						log.Debug("Balance Hit")
						sFloat := fmt.Sprintf("%s", val)
						for i, value := range sFloat {
							if strings.ContainsAny(string(value), ".") && (i+5) <= len(sFloat) {
								sFloat = sFloat[0:i+1] + sFloat[i+1:i+5]
								break
							}
						}
						sValue := fmt.Sprintf("%s %s", sFloat, "SWRM")
						log.Debugf("This is Main Balance: %s", sValue)
						SetDisplay("confirmedBalance", "innerHTML", sValue)
					}
				case "Settlement":
					{
						log.Debug("Settlement Hit")
						var settlement Settlement
						err = json.Unmarshal(val, &settlement)
						if err != nil {
							log.Error("Error Unmarshalling settlement: ", err.Error())
							return
						}
						log.Debug("This is Settlement: ", settlement)

						timeZone, err := time.LoadLocation("Local")
						if err != nil {
							log.Error("Error while loading Location in Settlement: ", err.Error())
							return
						}
						CurrentZone := (settlement.Date).In(timeZone)
						date := (CurrentZone).Format("02-01-2006")
						time := (CurrentZone).Format(time.Kitchen)
						sDateTime := fmt.Sprintf("%s %s", date, time)
						SetDisplay("NextDistribution", "innerHTML", sDateTime)
					}
				case "BalanceCycle":
					{
						log.Debug("BCN Hit")
						var bcnBalance BCNBalance
						err = json.Unmarshal(val, &bcnBalance)
						if err != nil {
							log.Error("Error in Unmarshalling BCN Balance:", err.Error())
							return
						}
						log.Debug("This is Balance Cycle: ", bcnBalance)
						sValue := fmt.Sprintf("%f %s", (bcnBalance.Owned - bcnBalance.Owe), "SWRM")
						SetDisplay("Pending", "innerHTML", sValue)
						SetDisplay("CycleDownloaded", "innerHTML", Humanize(bcnBalance.BytesDownloaded))
						SetDisplay("CycleServed", "innerHTML", Humanize(bcnBalance.BytesServed))
					}
				case "Peers":
					{
						log.Debug("Peers Hit")
						sValue := fmt.Sprintf("%s", val)
						log.Debugf("This is Number of Peers: %s", val)
						SetDisplay("PeersData", "innerHTML", sValue)
						GetPeers()
					}
				case "Settings":
					{
						log.Debug("Settings Hit")
						var settings Settings
						err = json.Unmarshal(val, &settings)
						if err != nil {
							log.Error("Error in Unmarshalling Settings: ", err.Error())
							return
						}
						log.Debug("This is Settings: ", settings)
						jsDoc := js.Global().Get("document")
						if !jsDoc.Truthy() {
							log.Error("Unable to get document object in settings")
							return
						}
						values := reflect.ValueOf(&settings).Elem()
						for key := 0; key < values.NumField(); key++ {
							name := values.Type().Field(key).Name
							value := values.Field(key).Interface()
							if (name == "MaxStorage") || (name == "UsedStorage") {
								OutputArea := jsDoc.Call("getElementById", name)
								if !OutputArea.Truthy() {
									log.Error("Unable to get output text area in settings keys")
									return
								}
								sValue := value
								if name == "MaxStorage" {
									sValue = fmt.Sprintf("%.2f %s", value, "GB")
								}
								if name == "UsedStorage" {
									sValue = fmt.Sprintf("%.2f %s", value, "GB")
								}
								OutputArea.Set("innerHTML", sValue)
							}
						}
					}
				default:
					{
						log.Debug("Default Hit")
					}
				}
			}
		}()
		return nil
	})
}

func Humanize(value float64) string {
	var rVal string
	switch true {
	case (value > 1073741823):
		{
			rVal = fmt.Sprintf("%.1f %s", (value / 1073741824), "GB")
		}
	case (value > 1048575):
		{
			rVal = fmt.Sprintf("%.1f %s", (value / 1048576), "MB")
		}
	case (value > 1023):
		{
			rVal = fmt.Sprintf("%.1f %s", (value / 1024), "KB")
		}
	default:
		{
			rVal = fmt.Sprintf("%.1f %s", value, "B")
		}
	}
	return rVal
}

func GetData(payload map[string]interface{}, funcName string) []uint8 {
	buf, err := json.Marshal(payload)
	if err != nil {
		log.Error("Error in marshalling payload in : ", funcName, err.Error())
		return nil
	}
	resp, err := http.Post(GATEWAY, "application/json", bytes.NewReader(buf))
	if err != nil {
		log.Error("Error in getting response in : ", funcName, err.Error())
		return nil
	}
	defer resp.Body.Close()
	respBuf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("Error in reading respBuf in : ", funcName, err.Error())
		return nil
	}
	data := make(map[string]string)
	err = json.Unmarshal(respBuf, &data)
	if err != nil {
		log.Error("Error in unmarshalling respbuf in : ", funcName, err.Error())
		return nil
	}
	log.Debug("This is data in : ", funcName, data["val"])
	var out Out
	err = json.Unmarshal([]byte(data["val"]), &out)
	if err != nil {
		log.Error("Error in unmarshalling data in : ", funcName, err.Error())
		return nil
	}
	val, err := json.Marshal(out.Data)
	if err != nil {
		log.Error("Error in marshalling out in : ", funcName, err.Error())
		return nil
	}
	return val
}

func ModifyConfig(payload map[string]interface{}, funcName string) (string, error) {
	buf, err := json.Marshal(payload)
	if err != nil {
		log.Error("Error in marshalling payload in : ", funcName, err.Error())
		return "", nil
	}
	resp, err := http.Post(GATEWAY, "application/json", bytes.NewReader(buf))
	if err != nil {
		log.Error("Error in getting response in : ", funcName, err.Error())
		return "", nil
	}
	defer resp.Body.Close()
	log.Debugf("This is response from %s  : ", funcName, resp)
	respBuf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("Error in reading respBuf in : ", funcName, err.Error())
		return "", nil
	}
	data := make(map[string]string)
	err = json.Unmarshal(respBuf, &data)
	if err != nil {
		log.Error("Error in unmarshalling respbuf in : ", funcName, err.Error())
		return "", nil
	}
	log.Debug(data["val"])
	return data["val"], nil
}

func SetDisplay(Id string, Attr string, value string) {
	for i := 0; i < 5; i++ {
		jsDoc := js.Global().Get("document")
		if !jsDoc.Truthy() {
			log.Error("Unable to get document object in: ", Id)
			return
		}
		OutputArea := jsDoc.Call("getElementById", Id)
		if !OutputArea.Truthy() {
			log.Error("Unable to get output area in: ", Id)
			log.Debugf("Trying to find OutputArea again in:%s ", Id)
			time.Sleep(1 * time.Second)
			continue
		} else {
			log.Debugf("OutputArea found in:%s ", Id)
			OutputArea.Set(Attr, value)
			break
		}
	}

}

func SetMultipleDisplay(Id string, Attributes map[string]string) {
	for i := 0; i < 5; i++ {
		jsDoc := js.Global().Get("document")
		if !jsDoc.Truthy() {
			log.Error("Unable to get document object in: ", Id)
			return
		}
		OutputArea := jsDoc.Call("getElementById", Id)
		if !OutputArea.Truthy() {
			log.Error("Unable to get output area in: ", Id)
			log.Debugf("Trying to find OutputArea again in:%s ", Id)
			time.Sleep(1 * time.Second)
			continue
		} else {
			log.Debugf("OutputArea found in:%s ", Id)
			for attr, value := range Attributes {
				OutputArea.Set(attr, value)
			}
			break
		}
	}
	return
}

func GetValue(Id string, Attr string) string {
	jsDoc := js.Global().Get("document")
	if !jsDoc.Truthy() {
		log.Error("Unable to get document object in: ", Id)
		return ""
	}
	OutputArea := jsDoc.Call("getElementById", Id)
	if !OutputArea.Truthy() {
		log.Error("Unable to get output area in: ", Id)
		log.Debugf("Trying to find OutputArea again in:%s ", Id)
		time.Sleep(1 * time.Second)
	} else {
		log.Debugf("OutputArea found in:%s ", Id)
		return fmt.Sprintf("%s", OutputArea.Get(Attr))
	}
	return ""
}

func CreateElement(Id string, element string, Attr string, value string) {
	jsDoc := js.Global().Get("document")
	if !jsDoc.Truthy() {
		log.Error("Unable to get document object in: ", Id)
		return
	}
	OutputArea := jsDoc.Call("createElement", element)
	if !OutputArea.Truthy() {
		log.Error("Unable to create div in: ", Id)
		return
	}
	if value != "" {
		OutputArea.Set(Attr, value)
	}
	jsDoc.Call("getElementById", Id).Call("appendChild", OutputArea)
}

func GetID() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			payload := map[string]interface{}{
				"val": strings.Join([]string{"hive-cli.exe", "id", "-j"}, splicer),
			}
			val := GetData(payload, "GetID")
			var id ID
			err := json.Unmarshal(val, &id)
			if err != nil {
				log.Error("Error in Unmarshalling ID in GetID: ", err.Error())
				return
			}
			SetDisplay("Address", "innerHTML", "")
			for _, value := range id.Addresses {
				CreateElement("Address", "div", "innerHTML", value)
				CreateElement("Address", "br", "innerHTML", "")
			}
			SetDisplay("PeerID", "innerHTML", id.PeerID)
		}()
		return nil
	})
}

func GetPeers() {
	payload := map[string]interface{}{
		"val": strings.Join([]string{"hive-cli.exe", "swarm", "peers", "-j"}, splicer),
	}
	val := GetData(payload, "GetPeers")
	var swarmPeers []string
	err := json.Unmarshal(val, &swarmPeers)
	if err != nil {
		log.Error("Error in Unmarshalling SwarmPeers: ", err.Error())
		return
	}
	SetDisplay("Peers", "innerHTML", "")
	for _, value := range swarmPeers {
		CreateElement("Peers", "div", "innerHTML", value)
		CreateElement("Peers", "br", "innerHTML", "")
	}
}

func SetEarningDropDown() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			payload := map[string]interface{}{
				"val": strings.Join([]string{"hive-cli.exe", "earning", "-g", "-j"}, splicer),
			}
			val := GetData(payload, "SetEarningDropDown")
			log.Debug("Earning Hit")
			var netEarnings NetEarnings
			err := json.Unmarshal(val, &netEarnings)
			if err != nil {
				log.Error("Error in Unmarshalling Net Earnings: ", err.Error())
				return
			}
			log.Debugf("%+v", netEarnings)
			jsDoc := js.Global().Get("document")
			if !jsDoc.Truthy() {
				log.Error("Unable to get document object in SetEarningDropDown")
				return
			}
			OutputArea := jsDoc.Call("getElementById", "DevicesDropDown")
			if !OutputArea.Truthy() {
				log.Error("Unable to get DevicesDropDown in SetEarningDropDown ")
				return
			}
			OutputArea = jsDoc.Call("createElement", "option")
			if !OutputArea.Truthy() {
				log.Error("Unable to get create option in DevicesDropDown")
				return
			}
			OutputArea.Set("innerHTML", "ALL DEVICES")
			OutputArea.Set("value", "ALL DEVICES")
			OutputArea.Set("selected", "true")
			jsDoc.Call("getElementById", "DevicesDropDown").Call("appendChild", OutputArea)
			for _, value := range netEarnings.Devices {
				OutputArea := jsDoc.Call("createElement", "option")
				if !OutputArea.Truthy() {
					log.Error("Unable to get create option in DevicesDropDown")
					return
				}
				sOption := fmt.Sprintf("%s-%s", value.PeerId, value.Name)
				OutputArea.Set("innerHTML", sOption)
				OutputArea.Set("value", value.PeerId)
				jsDoc.Call("getElementById", "DevicesDropDown").Call("appendChild", OutputArea)
			}
			log.Debugf("This is Device Total: %+v ", netEarnings.DeviceTotal)
		}()
		return nil
	})
}

func GetStorageLocation() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			payload := map[string]interface{}{
				"val": strings.Join([]string{"hive-cli.exe", "config", "get-storage-location", "-j"}, splicer),
			}
			buf, err := json.Marshal(payload)
			if err != nil {
				log.Error("Error in Marshalling Payload in GetStorageLocation: ", err.Error())
				return
			}
			resp, err := http.Post(GATEWAY, "application/json", bytes.NewReader(buf))
			if err != nil {
				log.Error("Error in getting response in GetStorageLocation: ", err.Error())
				return
			}
			defer resp.Body.Close()
			respBuf, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Error("Error in Reading Body in GetStorageLocation: ", err.Error())
				return
			}
			data := make(map[string]string)
			err = json.Unmarshal(respBuf, &data)
			if err != nil {
				log.Error("Error in Unmarshalling respBuf in GetStorageLocation: ", err.Error())
				return
			}
			var out Out
			err = json.Unmarshal([]byte(data["val"]), &out)
			if err != nil {
				log.Error("Error in Unmarshalling data in GetStorageLocation: ", err.Error())
				return
			}
			value := fmt.Sprintf("%s", out.Data)
			SetDisplay("StoragePath", "innerHTML", value)

		}()
		return nil
	})
}

func GetProfile() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			payload := map[string]interface{}{
				"val": strings.Join([]string{"hive-cli.exe", "profile", "-j"}, splicer),
			}
			val := GetData(payload, "GetProfile")
			var profile Profile
			err := json.Unmarshal(val, &profile)
			if err != nil {
				log.Error("Error in unmarshalling val in GetProfile: ", err.Error())
				return
			}
			SetDisplay("Email", "innerHTML", profile.Email)
			SetDisplay("Role", "innerHTML", profile.Role)
		}()
		return nil
	})
}

func GetBandwidth() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			payload := map[string]interface{}{
				"val": strings.Join([]string{"hive-cli.exe", "stat", "bandwidth", "-j"}, splicer),
			}
			val := GetData(payload, "GetBandwidth")
			var bandwidth Bandwidth
			err := json.Unmarshal(val, &bandwidth)
			if err != nil {
				log.Error("Error in unmarshalling val in GetBandwidth: ", err.Error())
				return
			}
			SetDisplay("Incoming", "innerHTML", Humanize(bandwidth.Incoming))

			SetDisplay("Outgoing", "innerHTML", Humanize(bandwidth.Outgoing))
		}()
		return nil
	})
}

func GetEarning() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resolve := args[0]
			go func() {
				payload := map[string]interface{}{
					"val": strings.Join([]string{"hive-cli.exe", "earning", "-g", "-j"}, splicer),
				}
				buf, err := json.Marshal(payload)
				if err != nil {
					log.Error("Error in marshalling Payload in GetEarning: ", err.Error())
					return
				}
				resp, err := http.Post(GATEWAY, "application/json", bytes.NewReader(buf))
				if err != nil {
					log.Error("Error in getting response in GetEarning: ", err.Error())
					return
				}
				defer resp.Body.Close()
				respBuf, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Error("Error in reading Body in GetEarning: ", err.Error())
					return
				}
				data := make(map[string]string)
				err = json.Unmarshal(respBuf, &data)
				if err != nil {
					log.Error("Error in unmarshalling respBuf in GetEarning: ", err.Error())
					return
				}
				var out Out
				err = json.Unmarshal([]byte(data["val"]), &out)
				if err != nil {
					log.Error("Error in unmarshalling data in GetEarning: ", err.Error())
					return
				}
				val, err := json.Marshal(out.Data)
				if err != nil {
					log.Error("Error in marshalling out in GetEarning: ", err.Error())
					return
				}
				var netEarnings NetEarnings
				err = json.Unmarshal(val, &netEarnings)
				if err != nil {
					log.Error("Error in Unmarshalling Net Earnings in GetEarning: ", err.Error())
					return
				}
				log.Debug("Sending details to CreateGraph from GetEarning")
				resolve.Invoke(data["val"])
			}()
			return nil
		})
		promiseConstructor := js.Global().Get("Promise")
		return promiseConstructor.New(handler)
		return nil
	})
}

func GetUptime() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			if StartTime != 0 {
				Start := time.Unix(StartTime, 0)
				elapsed := time.Since(Start)
				elapsed = elapsed.Round(time.Second)
				SetDisplay("Time", "innerHTML", fmt.Sprintf("%s", durafmt.Parse(elapsed)))
			}
		}()
		return nil
	})
}

func GetVersion() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			payload := map[string]interface{}{
				"val": strings.Join([]string{"hive-cli.exe", "version", "-j"}, splicer),
			}
			val := GetData(payload, "GetVersion")
			var version Version
			err := json.Unmarshal(val, &version)
			if err != nil {
				log.Error("Error in unmarshalling val in GetVersion: ", err.Error())
				return
			}
			SetDisplay("Version", "innerHTML", version.AppVersion)
		}()
		return nil
	})
}

func main() {
	logger.SetLogLevel("*", "Error")
	js.Global().Set("SetSwrmPortNumber", SetSwrmPortNumber())
	js.Global().Set("SetWebsocketPortNumber", SetWebsocketPortNumber())
	js.Global().Set("GetSettings", GetSettings())
	js.Global().Set("ModifyStorageSize", ModifyStorageSize())
	js.Global().Set("GetStatus", GetStatus())
	js.Global().Set("GetConfig", GetConfig())
	js.Global().Set("VerifyPort", VerifyPort())
	js.Global().Set("SetEarningDropDown", SetEarningDropDown())
	js.Global().Set("GetVersion", GetVersion())
	js.Global().Set("GetProfile", GetProfile())
	js.Global().Set("GetUptime", GetUptime())
	js.Global().Set("GetBandwidth", GetBandwidth())
	js.Global().Set("GetStorageLocation", GetStorageLocation())
	js.Global().Set("GetID", GetID())
	js.Global().Set("GetEarning", GetEarning())
	js.Global().Set("Events", Events())
	<-make(chan bool)
}
