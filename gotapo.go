// Package gotapo working
// with camera tapo like c200, c310
// by http
package gotapo

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// helper func
var p = fmt.Println

const (
	// MethodGet as link to methods
	MethodGet = "get"

	// MethodSet as link to methods
	MethodSet = "set"

	// MethodMR as link to methods
	MethodMR = "multipleRequest"

	// MethodDo as link to methods
	MethodDo = "do"

	// MethodLogin as link to methods
	MethodLogin = "login"

	// LastFileName for naming file with last preset
	LastFileName = "HereLastPreset"
)

var (

	// DefXBool need for Postfix xBool
	DefXBool = "Def"
)

// Types type of unnormal get bool value
type Types struct {
	Default   string
	Head      string
	Next      string
	isBinary  bool
	isCommand bool
	isSpecial string
	isTrue    bool
	isFalse   bool
}

// Stages type of unnormal get bool value. (with vals)
type Stages struct {
	Next     string `🟢:"🔴" 🔴:"🟢" +:"-" -:"+" on:"off" off:"on"`
	Binary   string `-:"true" +:"true" 🟢:"true" 🔴:"true" on:"true" off:"true"`
	True     string `+:"true" 🟢:"true" on:"true"`
	False    string `-:"true" 🔴:"true" off:"true"`
	Command  string `!:"true" @:"true" 🟢:"true" 🔴:"true" 🔵:"true" -:"true" +:"true" /:"true"`
	Special  string `!:"json" @:"exec"`
	sBool1   string `1:"🟢" 0:"🔴" 2:"🔵"`
	sBool2   string `1:"+" 0:"-" 2:"/"`
	sBoolDef string `1:"on" 0:"off" 2:"/"`
}

// presets type of Id Presets of cam
type presets struct {
	ID   string
	Name string
}

// elements type of Elements of cam
type elements struct {
	NightMode        *child
	NightModeAuto    *child
	PrivacyMode      *child
	Indicator        *child
	DetectMode       *child
	AutotrackingMode *child
	AlarmMode        *child
	ImageCorrection  *child
	ImageFlip        *child
}

// settings type of Settings of cam
type settings struct {
	PrintImageSettings         *child
	PrintImageSettings2        *child
	Time                       *child
	PresetChangeOsd            *child
	VisibleOsdTime             *child
	VisibleOsdText             *child
	OsdText                    string
	DetectSensitivity          int
	DetectSoundAlternativeMode *child
	DetectEnableSound          *child
	DetectEnableFlash          *child
}

// child assignment of function
type child struct {
	Value bool
	run   func()
}

// Tapo is general type with Vals
type Tapo struct {
	Parameters     map[string]string
	Host           string
	Port           string
	UserDef        string
	User           string
	Password       string
	UserID         string
	Rotate         bool
	FishEye        bool
	Flip           bool
	stokID         string
	TimeStr        string
	userGroup      string
	hashedPassword string
	hostURL        string
	deviceModel    string
	deviceID       string
	presets        []*presets
	lastPosition   int
	LastFile       string
	Elements       *elements
	Settings       *settings
	NextPreset     func()
	Reboot         func()
}

// Action is general Action cam
type Action interface {
	On()
	Off()
}

// updateStok type for upd key
type updateStok struct {
	Method string `json:"method"`
	Params struct {
		Hashed   bool   `json:"hashed"`
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"params"`
}

// updateStokReturn type for upd key return
type updateStokReturn struct {
	ErrorCode int64 `json:"error_code"`
	Result    struct {
		Stok      string `json:"stok"`
		UserGroup string `json:"user_group"`
	} `json:"result"`
}

// device type for device cam
type device struct {
	Method     string `json:"method"`
	DeviceInfo struct {
		Name []string `json:"name"`
	} `json:"device_info"`
}

// deviceRet type for device cam return
type deviceRet struct {
	DeviceInfo struct {
		BasicInfo struct {
			Barcode     string `json:"barcode"`
			DevID       string `json:"dev_id"`
			DeviceAlias string `json:"device_alias"`
			DeviceInfo  string `json:"device_info"`
			DeviceModel string `json:"device_model"`
			DeviceName  string `json:"device_name"`
			DeviceType  string `json:"device_type"`
			Features    string `json:"features"`
			HwDesc      string `json:"hw_desc"`
			HwVersion   string `json:"hw_version"`
			Mac         string `json:"mac"`
			OemID       string `json:"oem_id"`
			SwVersion   string `json:"sw_version"`
		} `json:"basic_info"`
	} `json:"device_info"`
	ErrorCode int64 `json:"error_code"`
}

// movePosition type for moving cam
type movePosition struct {
	Method string `json:"method"`
	Motor  struct {
		Move struct {
			YCoord string `json:"y_coord"`
			XCoord string `json:"x_coord"`
		} `json:"move"`
	} `json:"motor"`
}

// presetList type for moving presets cam
type presetList struct {
	Method string `json:"method"`
	Preset struct {
		Name []string `json:"name"`
	} `json:"preset"`
}

// presetListReturn type for moving presets cam return
type presetListReturn struct {
	ErrorCode int64 `json:"error_code"`
	Preset    struct {
		Preset struct {
			ID           []string `json:"id"`
			Name         []string `json:"name"`
			PositionPan  []string `json:"position_pan"`
			PositionTilt []string `json:"position_tilt"`
			ReadOnly     []string `json:"read_only"`
		} `json:"preset"`
	} `json:"preset"`
}

// nextPreset type for moving next preset
type nextPreset struct {
	Method string `json:"method"`
	Preset struct {
		GotoPreset struct {
			ID string `json:"id"`
		} `json:"goto_preset"`
	} `json:"preset"`
}

// reboot type for rebooting
type reboot struct {
	Method string `json:"method"`
	System struct {
		Reboot string `json:"reboot"`
	} `json:"system"`
}

// alarm type for alarming
type alarm struct {
	Method   string `json:"method"`
	MsgAlarm struct {
		Chn1MsgAlarmInfo struct {
			AlarmType string   `json:"alarm_type"`
			Enabled   string   `json:"enabled"`
			LightType string   `json:"light_type"`
			AlarmMode []string `json:"alarm_mode"`
		} `json:"chn1_msg_alarm_info"`
	} `json:"msg_alarm"`
}

// led type for led state
type led struct {
	Method string `json:"method"`
	Led    struct {
		Config struct {
			Enabled string `json:"enabled"`
		} `json:"config"`
	} `json:"led"`
}

// detect type for detect state
type detect struct {
	Method          string `json:"method"`
	MotionDetection struct {
		MotionDet struct {
			Enabled            string `json:"enabled"`
			DigitalSensitivity string `json:"digital_sensitivity"`
		} `json:"motion_det"`
	} `json:"motion_detection"`
}

// type privacy state
type privacy struct {
	Method   string `json:"method"`
	LensMask struct {
		LensMaskInfo struct {
			Enabled string `json:"enabled"`
		} `json:"lens_mask_info"`
	} `json:"lens_mask"`
}

// type nightmode state
type nightMode struct {
	Method string `json:"method"`
	Image  struct {
		Common struct {
			InfType string `json:"inf_type"`
		} `json:"common"`
	} `json:"image"`
}

// type autotracking state
type autotracking struct {
	Method      string `json:"method"`
	TargetTrack struct {
		TargetTrackInfo struct {
			Enabled string `json:"enabled"`
		} `json:"target_track_info"`
	} `json:"target_track"`
}

// type get settings
type getImageSettings struct {
	Method string `json:"method"`
	Image  struct {
		Name string `json:"name"`
	} `json:"image"`
}

// type get settings return
type getImageSettingsRet struct {
	Image struct {
		Switch struct {
			Name              string `json:".name"`
			Type              string `json:".type"`
			SwitchMode        string `json:"switch_mode"`
			ScheduleStartTime string `json:"schedule_start_time"`
			ScheduleEndTime   string `json:"schedule_end_time"`
			RotateType        string `json:"rotate_type"`
			FlipType          string `json:"flip_type"`
			Ldc               string `json:"ldc"`
			NightVisionMode   string `json:"night_vision_mode"`
			WtlIntensityLevel string `json:"wtl_intensity_level"`
		} `json:"switch"`
	} `json:"image"`
	ErrorCode int `json:"error_code"`
}

// type image settings fish eye
type setImageCorrection struct {
	Method string `json:"method"`
	Image  struct {
		Switch struct {
			Ldc string `json:"ldc"`
		} `json:"switch"`
	} `json:"image"`
}

// type flip
type setImageFlip struct {
	Method string `json:"method"`
	Image  struct {
		Switch struct {
			FlipType string `json:"flip_type"`
		} `json:"switch"`
	} `json:"image"`
}

// type get Time
type getTime struct {
	Method string `json:"method"`
	System struct {
		Name []string `json:"name"`
	} `json:"system"`
}

// type get Time return
type getTimeRet struct {
	System struct {
		ClockStatus struct {
			SecondsFrom1970 int    `json:"seconds_from_1970"`
			LocalTime       string `json:"local_time"`
		} `json:"clock_status"`
	} `json:"system"`
	ErrorCode int `json:"error_code"`
}

// type working with OSD
type getOSD struct {
	Method string `json:"method"`
	OSD    struct {
		Name  []string `json:"name"`
		Table []string `json:"table"`
	} `json:"OSD"`
}

// type working with OSD return
type getOSDRet struct {
	OSD struct {
		Date struct {
			Name    string `json:".name"`
			Type    string `json:".type"`
			XCoor   string `json:"x_coor"`
			YCoor   string `json:"y_coor"`
			Enabled string `json:"enabled"`
		} `json:"date"`
		Week struct {
			Name    string `json:".name"`
			Type    string `json:".type"`
			Enabled string `json:"enabled"`
			XCoor   string `json:"x_coor"`
			YCoor   string `json:"y_coor"`
		} `json:"week"`
		Font struct {
			Name      string `json:".name"`
			Type      string `json:".type"`
			Display   string `json:"display"`
			Size      string `json:"size"`
			ColorType string `json:"color_type"`
			Color     string `json:"color"`
		} `json:"font"`
		LabelInfo []struct {
			LabelInfo1 struct {
				Name    string `json:".name"`
				Type    string `json:".type"`
				XCoor   string `json:"x_coor"`
				YCoor   string `json:"y_coor"`
				Enabled string `json:"enabled"`
				Text    string `json:"text"`
			} `json:"label_info_1,omitempty"`
			LabelInfo2 struct {
				Name    string `json:".name"`
				Type    string `json:".type"`
				Enabled string `json:"enabled"`
				Text    string `json:"text"`
				XCoor   string `json:"x_coor"`
				YCoor   string `json:"y_coor"`
			} `json:"label_info_2,omitempty"`
			LabelInfo3 struct {
				Name    string `json:".name"`
				Type    string `json:".type"`
				Enabled string `json:"enabled"`
				Text    string `json:"text"`
				XCoor   string `json:"x_coor"`
				YCoor   string `json:"y_coor"`
			} `json:"label_info_3,omitempty"`
		} `json:"label_info"`
	} `json:"OSD"`
	ErrorCode int `json:"error_code"`
}

// type working with OSD
type osd struct {
	Method string `json:"method"`
	OSD    struct {
		Date struct {
			Enabled string `json:"enabled"`
			XCoor   int    `json:"x_coor"`
			YCoor   int    `json:"y_coor"`
		} `json:"date"`
		Week struct {
			Enabled string `json:"enabled"`
			XCoor   int    `json:"x_coor"`
			YCoor   int    `json:"y_coor"`
		} `json:"week"`
		Font struct {
			Color     string `json:"color"`
			ColorType string `json:"color_type"`
			Display   string `json:"display"`
			Size      string `json:"size"`
		} `json:"font"`
		LabelInfo1 struct {
			Enabled string `json:"enabled"`
			Text    string `json:"text"`
			XCoor   int    `json:"x_coor"`
			YCoor   int    `json:"y_coor"`
		} `json:"label_info_1"`
	} `json:"OSD"`
}

// nil func
func fnil() {
}

// Firsty initialise
func (o *Tapo) init() {
	h := md5.New()
	io.WriteString(h, o.Password)
	o.hashedPassword = strings.ToUpper(fmt.Sprintf("%x", h.Sum(nil)))
	newParam := make(map[string]string)
	o.Parameters = newParam
	o.Parameters["Host"] = o.Host
	o.Parameters["Referer"] = "https://" + o.Host + ":" + o.Port
	o.Parameters["Accept"] = "application/json"
	o.Parameters["Accept-Encoding"] = "gzip, deflate"
	o.Parameters["User-Agent"] = "Tapo CameraClient Android"
	o.Parameters["Connection"] = "close"
	o.Parameters["requestByApp"] = "true"
	o.Parameters["Content-Type"] = "application/json; charset=UTF-8"

	o.Settings = new(settings)
	o.Elements = new(elements)

	o.Settings.VisibleOsdTime = new(child)
	o.Settings.VisibleOsdTime.Value = true
	o.Settings.VisibleOsdTime.run = o.setOsd

	o.Settings.VisibleOsdText = new(child)
	o.Settings.VisibleOsdText.Value = false
	o.Settings.VisibleOsdText.run = o.setOsd

	o.Settings.OsdText = ""

	o.Elements.PrivacyMode = new(child)
	o.Elements.PrivacyMode.Value = false
	o.Elements.PrivacyMode.run = o.setPrivacy

	o.Elements.NightModeAuto = new(child)
	o.Elements.NightModeAuto.Value = true
	o.Elements.NightModeAuto.run = o.setNightModeAuto

	o.Elements.NightMode = new(child)
	o.Elements.NightMode.Value = true
	o.Elements.NightMode.run = o.setNightMode

	o.Elements.Indicator = new(child)
	o.Elements.Indicator.Value = true
	o.Elements.Indicator.run = o.setLed

	o.Elements.AutotrackingMode = new(child)
	o.Elements.AutotrackingMode.Value = false
	o.Elements.AutotrackingMode.run = o.setAutotracking

	o.Settings.PresetChangeOsd = new(child)
	o.Settings.PresetChangeOsd.Value = false
	o.Settings.PresetChangeOsd.run = fnil

	o.Elements.DetectMode = new(child)
	o.Elements.DetectMode.Value = false
	o.Elements.DetectMode.run = o.setDetect

	o.Settings.DetectSensitivity = 1

	o.Settings.DetectSoundAlternativeMode = new(child)
	o.Settings.DetectSoundAlternativeMode.Value = false
	o.Settings.DetectSoundAlternativeMode.run = o.setDetect

	o.Settings.DetectEnableSound = new(child)
	o.Settings.DetectEnableSound.Value = true
	o.Settings.DetectEnableSound.run = o.setDetect

	o.Settings.DetectEnableFlash = new(child)
	o.Settings.DetectEnableFlash.Value = false
	o.Settings.DetectEnableFlash.run = o.setDetect

	o.Elements.AlarmMode = new(child)
	o.Elements.AlarmMode.Value = false
	o.Elements.AlarmMode.run = o.setAlarm

	o.Settings.Time = new(child)
	o.Settings.Time.Value = true
	o.Settings.Time.run = o.getTime

	o.Settings.PrintImageSettings = new(child)
	o.Settings.PrintImageSettings.Value = true
	o.Settings.PrintImageSettings.run = o.getImageSettings

	o.Settings.PrintImageSettings2 = new(child)
	o.Settings.PrintImageSettings2.Value = true
	o.Settings.PrintImageSettings2.run = o.getImageSettings2

	o.Elements.ImageCorrection = new(child)
	o.Elements.ImageCorrection.Value = true
	o.Elements.ImageCorrection.run = o.setImageCorrection

	o.Elements.ImageFlip = new(child)
	o.Elements.ImageFlip.Value = true
	o.Elements.ImageFlip.run = o.setImageFlip

	o.NextPreset = o.setNextPreset
	o.Reboot = o.rebootDevice

	o.getImageSettings()
}

// POST query to cam
func (o *Tapo) query(data []byte) []byte {
	body := bytes.NewReader(data)
	req, _ := http.NewRequest("POST", o.hostURL, body)
	for k, v := range o.Parameters {
		req.Header.Add(k, v)
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return []byte{}
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}
	}
	defer resp.Body.Close()
	return b
}

// Refresh stok. For authentication
func (o *Tapo) update() {
	o.hostURL = `https://` + o.Host + `:` + o.Port
	bodyStruct := new(updateStok)
	result := new(updateStokReturn)
	bodyStruct.Method = MethodLogin
	bodyStruct.Params.Hashed = true
	bodyStruct.Params.Username = o.User
	bodyStruct.Params.Password = o.hashedPassword
	data, _ := json.Marshal(bodyStruct)
	json.Unmarshal(o.query(data), &result)
	o.stokID = result.Result.Stok
	o.userGroup = result.Result.UserGroup
	o.hostURL += `/stok=` + o.stokID + `/ds`
	if result.ErrorCode != 0 {
		p(`Authenticate failed. Try use another cred. login - "admin", password - your password in Tapo account.`)
		if o.UserDef != o.User {
			o.UserDef = o.User
			o.User = "admin"
			p(`App will be trying with "admin" with next operation(if your pass on rtsp equal pass your account). If next operation exist.`)
		} else {
			p(`Authenticate failed. Login by "admin" has failed.`)
			panic("!end operation authenticate!")
		}
	}
	o.getTime()
}

// Get information about device tapo c200
func (o *Tapo) getDevice() {
	o.update()
	bodyStruct := new(device)
	result := new(deviceRet)
	bodyStruct.Method = MethodGet
	bodyStruct.DeviceInfo.Name = []string{"basic_info"}
	data, _ := json.Marshal(bodyStruct)
	json.Unmarshal(o.query(data), &result)
	o.deviceModel = result.DeviceInfo.BasicInfo.DeviceModel
	o.deviceID = result.DeviceInfo.BasicInfo.DevID
}

// Manual move
//  10 = 10 degree
// -10 = 10 degree reverse
func (o *Tapo) setMovePosition(x int, y int) {
	o.update()
	bodyStruct := new(movePosition)
	bodyStruct.Method = MethodDo
	bodyStruct.Motor.Move.XCoord = strconv.Itoa(x)
	bodyStruct.Motor.Move.YCoord = strconv.Itoa(y)
	data, _ := json.Marshal(bodyStruct)
	o.query(data)
}

// Get all making Presets in App
func (o *Tapo) getPresets() {
	if o.Rotate {
		o.update()
		bodyStruct := new(presetList)
		result := new(presetListReturn)
		bodyStruct.Method = MethodGet
		bodyStruct.Preset.Name = []string{"preset"}
		data, _ := json.Marshal(bodyStruct)
		json.Unmarshal(o.query(data), &result)
		for i, v := range result.Preset.Preset.ID {
			pos := new(presets)
			pos.ID = v
			pos.Name = result.Preset.Preset.Name[i]
			o.presets = append(o.presets, pos)

		}
	}
}

// Switch to next preset
func (o *Tapo) setNextPreset() {
	if o.Rotate {
		o.update()
		o.lastPosition = o.rLast()
		max := new(presets)
		max.ID = "-9999"
		min := new(presets)
		min.ID = "9999"
		next := new(presets)
		next.ID = "0"
		check := true
		for _, v := range o.presets {
			vID, _ := strconv.Atoi(v.ID)
			minID, _ := strconv.Atoi(min.ID)
			maxID, _ := strconv.Atoi(max.ID)
			if vID < minID {
				min = v
			}
			if vID > maxID {
				max = v
			}
			if vID > o.lastPosition && check {
				next = v
				check = false
			}
		}
		if next.ID == "0" {
			next = min
			o.wLast(min.ID)
		}
		bodyStruct := new(nextPreset)
		bodyStruct.Method = MethodDo
		bodyStruct.Preset.GotoPreset.ID = next.ID
		data, _ := json.Marshal(bodyStruct)
		if o.Settings.PresetChangeOsd.Value {
			o.Settings.OsdText = next.Name
			o.Settings.VisibleOsdText.Value = true
			o.Settings.VisibleOsdTime.Value = true
			o.setOsd()
		}
		o.query(data)
		o.wLast(next.ID)
	}
}

// Write log last file
func (o *Tapo) wLast(text string) {
	ioutil.WriteFile(o.LastFile+`/`+LastFileName, []byte(text), 0775)
}

// Read log last file
func (o *Tapo) rLast() int {
	if lastDef, err := ioutil.ReadFile(o.LastFile + `/` + LastFileName); err == nil {
		last, _ := strconv.Atoi(string(lastDef))
		return last
	}
	os.Create(o.LastFile + `/` + LastFileName)
	return 0
}

// Run all presets with timer beetween
func (o *Tapo) runAllPresets(timer string) {
	if o.Rotate {
		durDef, _ := time.ParseDuration(timer)
		for range o.presets {
			time.Sleep(durDef)
			o.setNextPreset()
		}
	}
}

// Reboot device
func (o *Tapo) rebootDevice() {
	o.update()
	bodyStruct := new(reboot)
	bodyStruct.Method = MethodDo
	bodyStruct.System.Reboot = "null"
	data, _ := json.Marshal(bodyStruct)
	o.query(data)
}

// special function xBool
func (o *Types) xBool(s interface{}) *Types {
	switch s.(type) {
	case string:
		if _, err := strconv.Atoi(s.(string)); err == nil {
			ss := strings.Split(s.(string), "")
			sslen := strconv.Itoa(len(ss))
			ssss := ss[len(ss)-1]
			tmp, _ := reflect.TypeOf(*new(Stages)).FieldByName("sBool" + sslen)
			if val, ex := tmp.Tag.Lookup(ssss); ex {
				o.Default = val
			}
		} else {
			o.Default = s.(string)
			o.Head = o.Default
			o.test()
			return o
		}
	case bool:
		switch s.(bool) {
		case true:
			tmp, _ := reflect.TypeOf(*new(Stages)).FieldByName("sBool" + DefXBool)
			if val, ex := tmp.Tag.Lookup("1"); ex {
				o.Default = val
			}
		case false:
			tmp, _ := reflect.TypeOf(*new(Stages)).FieldByName("sBool" + DefXBool)
			if val, ex := tmp.Tag.Lookup("0"); ex {
				o.Default = val
			}
		}
	case int:
		ss := strings.Split(strconv.Itoa(s.(int)), "")
		sslen := strconv.Itoa(len(ss))
		ssss := ss[len(ss)-1]
		tmp, _ := reflect.TypeOf(*new(Stages)).FieldByName("sBool" + sslen)
		if val, ex := tmp.Tag.Lookup(ssss); ex {
			o.Default = val
		}
	}
	o.Head = strings.Split(o.Default, "")[0]
	o.test()
	return o
}

// special function xBool
func (o *Types) test() {
	tmp, _ := reflect.TypeOf(*new(Stages)).FieldByName("True")
	if val, ex := tmp.Tag.Lookup(o.Head); ex {
		b, _ := strconv.ParseBool(val)
		o.isTrue = b
	}
	tmp, _ = reflect.TypeOf(*new(Stages)).FieldByName("False")
	if val, ex := tmp.Tag.Lookup(o.Head); ex {
		b, _ := strconv.ParseBool(val)
		o.isFalse = b
	}
	tmp, _ = reflect.TypeOf(*new(Stages)).FieldByName("Binary")
	if val, ex := tmp.Tag.Lookup(o.Head); ex {
		b, _ := strconv.ParseBool(val)
		o.isBinary = b
	}
	tmp, _ = reflect.TypeOf(*new(Stages)).FieldByName("Command")
	if val, ex := tmp.Tag.Lookup(o.Head); ex {
		b, _ := strconv.ParseBool(val)
		o.isCommand = b
	}
	tmp, _ = reflect.TypeOf(*new(Stages)).FieldByName("isSpecial")
	if val, ex := tmp.Tag.Lookup(o.Head); ex {
		o.isSpecial = val
	}
	tmp, _ = reflect.TypeOf(*new(Stages)).FieldByName("Next")
	if val, ex := tmp.Tag.Lookup(o.Head); ex {
		o.Next = val
	}
}

func sBool(val interface{}) bool {
	switch val.(type) {
	case string:
		switch val.(string) {
		case "+":
			return true
		case "-":
			return false
		case "true":
			return true
		case "false":
			return false
		case "🔴":
			return false
		case "🟢":
			return true
		case "0":
			return false
		case "1":
			return true
		case "off":
			return false
		case "on":
			return true
		}
	case bool:
		return val.(bool)
	default:
		fmt.Println("Error!")
		return false
	}
	return false
}

// Take value
func addBoolArrString(arr []string, b bool, val string) []string {
	if b {
		arr = append(arr, val)
	}
	return arr
}

// Set alarm mode
// DetectEnableSound - include noise
// DetectSoundAlternativeMode - sound like a bip
// DetectEnableFlash - blinking led diode
func (o *Tapo) setAlarm() {
	o.update()
	bodyStruct := new(alarm)
	bodyStruct.Method = MethodSet
	if o.Settings.DetectSoundAlternativeMode.Value {
		bodyStruct.MsgAlarm.Chn1MsgAlarmInfo.AlarmType = "1"
	} else {
		bodyStruct.MsgAlarm.Chn1MsgAlarmInfo.AlarmType = "0"
	}
	bodyStruct.MsgAlarm.Chn1MsgAlarmInfo.LightType = "1"
	bodyStruct.MsgAlarm.Chn1MsgAlarmInfo.Enabled = new(Types).xBool(o.Elements.AlarmMode.Value).Default
	bodyStruct.MsgAlarm.Chn1MsgAlarmInfo.AlarmMode = addBoolArrString(bodyStruct.MsgAlarm.Chn1MsgAlarmInfo.AlarmMode, o.Settings.DetectEnableSound.Value, "sound")
	bodyStruct.MsgAlarm.Chn1MsgAlarmInfo.AlarmMode = addBoolArrString(bodyStruct.MsgAlarm.Chn1MsgAlarmInfo.AlarmMode, o.Settings.DetectEnableFlash.Value, "light")
	bodyStruct.MsgAlarm.Chn1MsgAlarmInfo.AlarmMode = addBoolArrString(bodyStruct.MsgAlarm.Chn1MsgAlarmInfo.AlarmMode, !(o.Settings.DetectEnableSound.Value && o.Settings.DetectEnableFlash.Value), "sound")
	data, _ := json.Marshal(bodyStruct)
	o.query(data)
}

// Turn Indicator diode (red, green)
func (o *Tapo) setLed() {
	o.update()
	bodyStruct := new(led)
	bodyStruct.Method = MethodSet
	bodyStruct.Led.Config.Enabled = new(Types).xBool(o.Elements.Indicator.Value).Default
	data, _ := json.Marshal(bodyStruct)
	o.query(data)

}

// Get Time
func (o *Tapo) getTime() {
	result := new(getTimeRet)
	bodyStruct := new(getTime)
	bodyStruct.Method = MethodGet
	bodyStruct.System.Name = []string{"clock_status"}
	data, _ := json.Marshal(bodyStruct)
	json.Unmarshal(o.query(data), &result)
	o.TimeStr = result.System.ClockStatus.LocalTime
}

// Get Settings Image
func (o *Tapo) getImageSettings() {
	o.update()
	bodyStruct := new(getImageSettings)
	bodyStruct.Method = MethodGet
	bodyStruct.Image.Name = "common"
	data, _ := json.Marshal(bodyStruct)
	o.query(data)
	o.getImageSettings2()
	o.getTime()
}

// Get Correction
func (o *Tapo) getImageSettings2() {
	o.update()
	result := new(getImageSettingsRet)
	bodyStruct := new(getImageSettings)
	bodyStruct.Method = MethodGet
	bodyStruct.Image.Name = "switch"
	data, _ := json.Marshal(bodyStruct)
	json.Unmarshal(o.query(data), &result)
	o.Rotate = new(Types).xBool(result.Image.Switch.RotateType).isTrue
	o.FishEye = new(Types).xBool(result.Image.Switch.Ldc).isTrue
	o.Flip = result.Image.Switch.FlipType == "center"
}

// Set Correction
func (o *Tapo) setImageCorrection() {
	o.update()
	bodyStruct := new(setImageCorrection)
	bodyStruct.Method = MethodSet
	bodyStruct.Image.Switch.Ldc = new(Types).xBool(o.Elements.ImageCorrection.Value).Default
	data, _ := json.Marshal(bodyStruct)
	o.query(data)
}

// Set Flip
func (o *Tapo) setImageFlip() {
	val := new(Types).xBool(o.Elements.ImageFlip.Value).Default
	if o.Elements.ImageFlip.Value {
		val = "center"
	}
	o.update()
	bodyStruct := new(setImageFlip)
	bodyStruct.Method = MethodSet
	bodyStruct.Image.Switch.FlipType = val
	data, _ := json.Marshal(bodyStruct)
	o.query(data)
}

// Motion detect with sensitivity
func (o *Tapo) setDetect() {
	o.update()
	bodyStruct := new(detect)
	bodyStruct.Method = MethodSet
	switch {
	case o.Settings.DetectSensitivity == 1:
		bodyStruct.MotionDetection.MotionDet.DigitalSensitivity = "20"
	case o.Settings.DetectSensitivity == 2:
		bodyStruct.MotionDetection.MotionDet.DigitalSensitivity = "50"
	case o.Settings.DetectSensitivity == 3:
		bodyStruct.MotionDetection.MotionDet.DigitalSensitivity = "80"
	default:
		bodyStruct.MotionDetection.MotionDet.DigitalSensitivity = "20"
	}
	bodyStruct.MotionDetection.MotionDet.Enabled = new(Types).xBool(o.Elements.DetectMode.Value).Default
	data, _ := json.Marshal(bodyStruct)
	o.query(data)
}

// Turn camera in private mode with stop video channel
func (o *Tapo) setPrivacy() {
	o.update()
	bodyStruct := new(privacy)
	bodyStruct.Method = MethodSet
	bodyStruct.LensMask.LensMaskInfo.Enabled = new(Types).xBool(o.Elements.PrivacyMode.Value).Default
	data, _ := json.Marshal(bodyStruct)
	o.query(data)
}

// Turn irc flashlight
func (o *Tapo) setNightMode() {
	o.update()
	bodyStruct := new(nightMode)
	bodyStruct.Method = MethodSet
	bodyStruct.Image.Common.InfType = new(Types).xBool(o.Elements.NightMode.Value).Default
	data, _ := json.Marshal(bodyStruct)
	o.query(data)
}

// Turn irc flashlight
func (o *Tapo) setNightModeAuto() {
	if o.Elements.NightModeAuto.Value {
		o.update()
		bodyStruct := new(nightMode)
		bodyStruct.Method = MethodSet
		bodyStruct.Image.Common.InfType = "auto"
		data, _ := json.Marshal(bodyStruct)
		o.query(data)
	}
}

// Autotracking all motion. BETA
func (o *Tapo) setAutotracking() {
	o.update()
	bodyStruct := new(autotracking)
	bodyStruct.Method = MethodSet
	bodyStruct.TargetTrack.TargetTrackInfo.Enabled = new(Types).xBool(o.Elements.AutotrackingMode.Value).Default
	data, _ := json.Marshal(bodyStruct)
	o.query(data)
}

// get Text OSD
func (o *Tapo) getOsd() {
	o.update()
	bodyStruct := new(getOSD)
	result := new(getOSDRet)
	bodyStruct.Method = MethodGet
	bodyStruct.OSD.Name = []string{"date", "week", "font"}
	bodyStruct.OSD.Table = []string{"label_info"}
	data, _ := json.Marshal(bodyStruct)
	json.Unmarshal(o.query(data), &result)
	o.Settings.OsdText = result.OSD.LabelInfo[0].LabelInfo1.Text
}

// Text OSD
func (o *Tapo) setOsd() {
	if len(o.Settings.OsdText) == 0 {
		o.getOsd()
	}
	if len(o.Settings.OsdText) > 16 {
		//16 symbols, not bytes
		o.Settings.OsdText = string([]rune(o.Settings.OsdText)[0:16])
	}
	o.update()
	bodyStruct := new(osd)
	bodyStruct.Method = MethodSet
	bodyStruct.OSD.Date.Enabled = new(Types).xBool(o.Settings.VisibleOsdTime.Value).Default
	bodyStruct.OSD.Date.XCoor = 0
	bodyStruct.OSD.Date.YCoor = 0
	bodyStruct.OSD.Font.Color = "white"
	bodyStruct.OSD.Font.ColorType = "auto"
	bodyStruct.OSD.Font.Display = "ntnb"
	bodyStruct.OSD.Font.Size = "auto"
	bodyStruct.OSD.LabelInfo1.Enabled = new(Types).xBool(o.Settings.VisibleOsdText.Value).Default
	bodyStruct.OSD.LabelInfo1.Text = o.Settings.OsdText
	bodyStruct.OSD.LabelInfo1.XCoor = 0
	bodyStruct.OSD.LabelInfo1.YCoor = 450
	//---china weeks---
	bodyStruct.OSD.Week.Enabled = new(Types).xBool(false).Default
	bodyStruct.OSD.Week.XCoor = 0
	bodyStruct.OSD.Week.YCoor = 0
	//---china weeks---
	data, _ := json.Marshal(bodyStruct)
	o.query(data)
}

// Connect is general function for connecting to Camera
func Connect(host string, user string, password string) *Tapo {
	o := new(Tapo)
	o.LastFile, _ = filepath.Abs(filepath.Dir(os.Args[0]))
	o.Host = host
	o.Port = "443"
	o.User = user
	o.Password = password
	o.init()
	o.getDevice()
	o.getPresets()

	return o
}

// On is turn settings
func (o *child) On() {
	o.Value = true
	o.run()
}

// Off is turn settings
func (o *child) Off() {
	o.Value = false
	o.run()
}

// On is turn settings
func (o *Tapo) On(s Action) {
	s.On()
}

// Off is turn settings
func (o *Tapo) Off(s Action) {
	s.Off()
}

// MoveRight is moving cam to right
func (o *Tapo) MoveRight(val int) {
	o.setMovePosition(val, 0)
	time.Sleep(5 * time.Second)
}

// MoveLeft is moving cam to left
func (o *Tapo) MoveLeft(val int) {
	o.setMovePosition(-val, 0)
	time.Sleep(5 * time.Second)
}

// MoveUp is moving cam to up
func (o *Tapo) MoveUp(val int) {
	o.setMovePosition(0, val)
	time.Sleep(5 * time.Second)
}

// MoveDown is moving cam to down
func (o *Tapo) MoveDown(val int) {
	o.setMovePosition(0, -val)
	time.Sleep(5 * time.Second)
}

// MoveTest is moving cam to all presets
func (o *Tapo) MoveTest() {
	o.runAllPresets("10s")
}
