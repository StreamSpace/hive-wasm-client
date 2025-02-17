// GOOS=js GOARCH=wasm go build -o  ../assets/hive.wasm
package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"syscall/js"
)

var (
	DNSState       bool
	DriveFreeSpace float64
)

func GetSettings() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			payload := map[string]interface{}{
				"val": strings.Join([]string{"hive-cli.exe", "settings", "-g", "-j"}, splicer),
			}
			log.Debug("Settings Hit")
			val := GetData(payload, "GetSettings")
			var settings Settings
			err := json.Unmarshal(val, &settings)
			if err != nil {
				log.Error("Error in unmarshalling val in GetSettings: ", err.Error())
				return
			}
			log.Debug(settings)
			SetDisplay("Name", "innerHTML", settings.Name)
			UsedSpace := settings.UsedStorage
			sUsedSpace := fmt.Sprintf("%.2f %s", UsedSpace*1024, "MB")
			SetDisplay("UsedSpace", "innerHTML", sUsedSpace)
			freeSpace := (settings.MaxStorage - settings.UsedStorage)
			sFreeSpace := fmt.Sprintf("%.2f %s", freeSpace*1024, "MB")
			SetDisplay("FreeSpace", "innerHTML", sFreeSpace)
			SetDisplay("StorageMin", "innerHTML", fmt.Sprintf("%.1f GB", UsedSpace))
			SetDisplay("rangeSlider", "min", fmt.Sprintf("%.1f", UsedSpace))
			DriveFreeSpace = settings.FreeDiskSpace / (1024 * 1024 * 1024)
			log.Debugf("Free Space in Drive: %.1f", DriveFreeSpace)
			SetDisplay("StorageMax", "innerHTML", fmt.Sprintf("%.1f GB", DriveFreeSpace))
			SetDisplay("rangeSlider", "max", fmt.Sprintf("%.1f", DriveFreeSpace))
			MaxStorage := fmt.Sprintf("%.1f", settings.MaxStorage)
			log.Debugf("MaxStorage: %s", MaxStorage)
			SetDisplay("rangeSlider", "value", MaxStorage)
			DNSState = settings.IsDNSEligible
		}()
		return nil
	})
}

func GetStatus() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			payload := map[string]interface{}{
				"val": strings.Join([]string{"hive-cli.exe", "status", "-j"}, splicer),
			}
			log.Debug("GetStatus Hit")
			val := GetData(payload, "GetStatus")
			var status Status
			err := json.Unmarshal(val, &status)
			if err != nil {
				log.Error("Error in unmarshalling val in GetStatus: ", err.Error())
				return
			}
			var sValue string
			if status.LoggedIn == true {
				sValue = "LoggedIn"
			} else if status.LoggedIn == false {
				sValue = "LoggedOut"
			}
			SetDisplay("LoggedIn", "innerHTML", sValue)
			StartTime = status.SessionStartTime
			CheckBanner()
		}()
		return nil
	})
}

func GetConfig() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			payload := map[string]interface{}{
				"val": strings.Join([]string{"hive-cli.exe", "config", "show", "-j"}, splicer),
			}
			log.Debug("GetConfig Hit")
			val := GetData(payload, "GetConfig")
			var config Config
			err := json.Unmarshal(val, &config)
			if err != nil {
				log.Error("Error in unmarshalling val in GetConfig: ", err.Error())
				return
			}
			log.Debug(config)
			SetDisplay("SwrmPortNumber", "placeholder", config.SwarmPort)
			Attributes := make(map[string]string)
			if DNSState == false {
				Attributes["style"] = "display: none;"
				Attributes["aria-hidden"] = "true"
				Attributes["visibility"] = "hidden"
				SetMultipleDisplay("Group_62_ID", Attributes)
				return
			}
			SetDisplay("WebSocketPortNumber", "placeholder", config.WebsocketPort)
		}()
		return nil
	})
}

func CheckBanner() {
	log.Debug("Checking Banner")
	localStorage := js.Global().Get("localStorage")
	if !localStorage.Truthy() {
		log.Error("Unable to get localStorage in CheckBanner")
		return
	}
	DaemonStartedAt := fmt.Sprintf("%s", localStorage.Get("DaemonStartedAt"))
	sStartTime := fmt.Sprintf("%d", StartTime)
	sRefreshState := fmt.Sprintf("%s", localStorage.Get("RefreshState"))
	log.Debugf("DaemonStartedAt: %s \n StartTime: %s", DaemonStartedAt, sStartTime)
	if sRefreshState == "Not Refreshed" {
		if sStartTime == DaemonStartedAt {
			SetDisplay("RestartBanner", "style", "display: block;")
			return
		} else if sStartTime != DaemonStartedAt {
			SetDisplay("RestartBanner", "style", "display: none;")
			localStorage.Set("DaemonStartedAt", StartTime)
			localStorage.Set("RefreshState", "Refreshed")
			return
		}
	}
	localStorage.Set("DaemonStartedAt", StartTime)
	localStorage.Set("RefreshState", "Refreshed")
}

func CheckPort(port string) (status bool, condition string) {
	if port == "" {
		return false, fmt.Sprintf("Enter A Valid Port Number")
	}
	val, err := strconv.Atoi(port)
	if err != nil {
		return false, fmt.Sprintf("Port %s is Not a Number", port)
	}
	if val < 1025 || val > 49150 {
		return false, fmt.Sprintf("Port %s is Unavailable", port)
	}
	return true, ""
}

func SaveSettings() {
	log.Debug("Saving Settings")
	payload := map[string]interface{}{
		"val": strings.Join([]string{"hive-cli.exe", "settings", "-j"}, splicer),
	}
	val, err := ModifyConfig(payload, "SaveSettings")
	if err != nil {
		log.Error("Error in Saving Settings")
		return
	}
	log.Debug("Settings Saved: ", val)
	return
}

func SetSwrmPortNumber() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			log.Debug("Updating SwarmPort Number")
			SetDisplay("SwrmPortStatus", "innerHTML", "")
			port := GetValue("SwrmPortNumber", "value")
			Attributes := make(map[string]string)
			status, condition := CheckPort(port)
			if status == true {
				payload := map[string]interface{}{
					"val": strings.Join([]string{"hive-cli.exe", "config", "modify", "SwarmPort", port}, splicer),
				}
				log.Debugf("Payload in SetSwrmPortNumber: %s", payload)
				val, _ := ModifyConfig(payload, "SetSwrmPortNumber")
				if strings.Contains(val, "not") {
					Attributes["innerHTML"] = fmt.Sprintf("Port %s is Unavailable", port)
					Attributes["style"] = "color: red;"
					SetMultipleDisplay("SwrmPortStatus", Attributes)
					return
				}
				SetDisplay("SwrmPortNumber", "placeholder", port)
				SetDisplay("RestartBanner", "style", "display: block;")
				Attributes["innerHTML"] = fmt.Sprintf("SwrmPort Changed to %s", port)
				Attributes["style"] = "color: #32CD32;"
				SetMultipleDisplay("SwrmPortStatus", Attributes)
				localStorage := js.Global().Get("localStorage")
				if !localStorage.Truthy() {
					log.Error("Unable to get localStorage in SwrmPortNumber")
					return
				}
				localStorage.Set("RefreshState", "Not Refreshed")
				return
			} else if status == false {
				Attributes["innerHTML"] = condition
				Attributes["style"] = "color: red;"
				SetMultipleDisplay("SwrmPortStatus", Attributes)
			}
		}()
		return nil
	})
}

func SetWebsocketPortNumber() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			log.Debug("Updating SetWebsocketPortNumber Number")
			SetDisplay("WebsocketPortStatus", "innerHTML", "")
			port := GetValue("WebSocketPortNumber", "value")
			Attributes := make(map[string]string)
			status, condition := CheckPort(port)
			if status == true {
				payload := map[string]interface{}{
					"val": strings.Join([]string{"hive-cli.exe", "config", "modify", "WebsocketPort", port}, splicer),
				}
				log.Debugf("Payload in SetWebsocketPortNumber: %s", payload)
				val, _ := ModifyConfig(payload, "SetWebsocketPortNumber")
				if strings.Contains(val, "not") {
					Attributes["innerHTML"] = fmt.Sprintf("Port %s is Unavailable", port)
					Attributes["style"] = "color: red;"
					SetMultipleDisplay("WebsocketPortStatus", Attributes)
					return
				}
				log.Debug("SwrmPort Updated Successfully")
				SetDisplay("WebSocketPortNumber", "placeholder", port)
				SetDisplay("RestartBanner", "style", "display: block;")
				Attributes["innerHTML"] = fmt.Sprintf("WebsocketPort Changed to %s", port)
				Attributes["style"] = "color: #32CD32;"
				SetMultipleDisplay("WebsocketPortStatus", Attributes)
				localStorage := js.Global().Get("localStorage")
				if !localStorage.Truthy() {
					log.Error("Unable to get localStorage in WebsocketPortNumber")
					return
				}
				localStorage.Set("RefreshState", "Not Refreshed")
				return
			} else if status == false {
				Attributes["innerHTML"] = condition
				Attributes["style"] = "color: red;"
				SetMultipleDisplay("WebsocketPortStatus", Attributes)
			}
		}()
		return nil
	})
}

func VerifyPort() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			log.Debug("Verifying Port Forwarding....")
			Attributes := make(map[string]string)
			Attributes["innerHTML"] = "Verifying...."
			Attributes["style"] = "color: rgba(219,219,219,1);"
			SetMultipleDisplay("PortForward", Attributes)
			payload := map[string]interface{}{
				"val": strings.Join([]string{"hive-cli.exe", "verify-port-forward"}, splicer),
			}
			val, err := ModifyConfig(payload, "VerifyPort")
			if err != nil {
				log.Error("Error in Checking Port Forwarding Status")
				SetDisplay("PortForward", "innerHTML", "Error in Checking")
				return
			}
			log.Debugf("This is val: %s", val)
			if strings.Contains(val, "NOT") {
				log.Debug("Port Forward Not Verified")
				Attributes["innerHTML"] = "Not Forwarded &#10008;"
				Attributes["style"] = "color: rgba(244,105,50,1);"
				SetMultipleDisplay("PortForward", Attributes)
				return
			}
			log.Debug("Port Forward Verified")
			Attributes["innerHTML"] = "Port Forwarded &#10004;"
			Attributes["style"] = "color: rgba(244,105,50,1);"
			SetMultipleDisplay("PortForward", Attributes)
		}()
		return nil
	})
}

func ModifyStorageSize() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			val := GetValue("rangeSlider", "value")
			log.Debug("Changing Storage Size to: ", val)
			payload := map[string]interface{}{
				"val": strings.Join([]string{"hive-cli.exe", "config", "modify", "Storage", val}, splicer),
			}
			log.Debug("Payload in Modify Storage Size: ", payload)
			val, err := ModifyConfig(payload, "ModifyStorageSize")
			if err != nil {
				log.Error("Error in Modifying Storage Size", err.Error())
				return
			}
			log.Debug("val in ModifyStorageSize: ", val)
			SaveSettings()
		}()
		return nil
	})
}
