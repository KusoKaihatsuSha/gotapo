// gotapo.go
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
	"strconv"
	"strings"
	"time"
)

var p = fmt.Println

const (
	MethodGet    = "get"
	MethodSet    = "set"
	MethodMR     = "multipleRequest"
	MethodDo     = "do"
	MethodLogin  = "login"
	LastFileName = "HereLastPreset"
)

type presets struct {
	Id   string
	Name string
}

type elements struct {
	NightMode        *Child
	NightModeAuto    *Child
	PrivacyMode      *Child
	Indicator        *Child
	DetectMode       *Child
	AutotrackingMode *Child
	AlarmMode        *Child
}

type settings struct {
	User                       *Child
	Time                       *Child
	PresetChangeOsd            *Child
	VisibleOsdTime             *Child
	VisibleOsdText             *Child
	OsdText                    string
	DetectSensitivity          int
	DetectSoundAlternativeMode *Child
	DetectEnableSound          *Child
	DetectEnableFlash          *Child
}

type Child struct {
	Value bool
	run   func()
}

type Tapo struct {
	Parameters     map[string]string
	Host           string
	Port           string
	User           string
	Password       string
	UserID         string
	stokId         string
	TimeStr        string
	userGroup      string
	hashedPassword string
	hostURL        string
	deviceModel    string
	deviceId       string
	presets        []*presets
	lastPosition   int
	LastFile       string
	Elements       *elements
	Settings       *settings
	NextPreset     func()
	Reboot         func()
}

type Action interface {
	On()
	Off()
}

type updateStok struct {
	Method string `json:"method"`
	Params struct {
		Hashed   bool   `json:"hashed"`
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"params"`
}

type updateStokReturn struct {
	ErrorCode int64 `json:"error_code"`
	Result    struct {
		Stok      string `json:"stok"`
		UserGroup string `json:"user_group"`
	} `json:"result"`
}

type device struct {
	Method     string `json:"method"`
	DeviceInfo struct {
		Name []string `json:"name"`
	} `json:"device_info"`
}

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

type MovePosition struct {
	Method string `json:"method"`
	Motor  struct {
		Move struct {
			YCoord string `json:"y_coord"`
			XCoord string `json:"x_coord"`
		} `json:"move"`
	} `json:"motor"`
}

type PresetList struct {
	Method string `json:"method"`
	Preset struct {
		Name []string `json:"name"`
	} `json:"preset"`
}

type PresetListReturn struct {
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

type NextPreset struct {
	Method string `json:"method"`
	Preset struct {
		GotoPreset struct {
			ID string `json:"id"`
		} `json:"goto_preset"`
	} `json:"preset"`
}

type reboot struct {
	Method string `json:"method"`
	System struct {
		Reboot string `json:"reboot"`
	} `json:"system"`
}

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

type led struct {
	Method string `json:"method"`
	Led    struct {
		Config struct {
			Enabled string `json:"enabled"`
		} `json:"config"`
	} `json:"led"`
}

type detect struct {
	Method          string `json:"method"`
	MotionDetection struct {
		MotionDet struct {
			Enabled            string `json:"enabled"`
			DigitalSensitivity string `json:"digital_sensitivity"`
		} `json:"motion_det"`
	} `json:"motion_detection"`
}

type privacy struct {
	Method   string `json:"method"`
	LensMask struct {
		LensMaskInfo struct {
			Enabled string `json:"enabled"`
		} `json:"lens_mask_info"`
	} `json:"lens_mask"`
}

type nightMode struct {
	Method string `json:"method"`
	Image  struct {
		Common struct {
			InfType string `json:"inf_type"`
		} `json:"common"`
	} `json:"image"`
}

type autotracking struct {
	Method      string `json:"method"`
	TargetTrack struct {
		TargetTrackInfo struct {
			Enabled string `json:"enabled"`
		} `json:"target_track_info"`
	} `json:"target_track"`
}

type getUser struct {
	Method    string `json:"method"`
	GetUserID struct {
		ID string `json:"id"`
	} `json:"get_user_id"`
}

type getTime struct {
	Method string `json:"method"`
	System struct {
		Name []string `json:"name"`
	} `json:"system"`
}

type getTimeRet struct {
	System struct {
		ClockStatus struct {
			SecondsFrom1970 int    `json:"seconds_from_1970"`
			LocalTime       string `json:"local_time"`
		} `json:"clock_status"`
	} `json:"system"`
	ErrorCode int `json:"error_code"`
}

type getOSD struct {
	Method string `json:"method"`
	OSD    struct {
		Name  []string `json:"name"`
		Table []string `json:"table"`
	} `json:"OSD"`
}

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

//----------------------------------------------------

func fnil() {
}

// Firsty initialise
func (o *Tapo) init() {
	h := md5.New()
	io.WriteString(h, o.Password)
	o.hashedPassword = strings.ToUpper(fmt.Sprintf("%x", h.Sum(nil)))
	new_param := make(map[string]string)
	o.Parameters = new_param
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

	o.Settings.VisibleOsdTime = new(Child)
	o.Settings.VisibleOsdTime.Value = true
	o.Settings.VisibleOsdTime.run = o.setOsd

	o.Settings.VisibleOsdText = new(Child)
	o.Settings.VisibleOsdText.Value = false
	o.Settings.VisibleOsdText.run = o.setOsd

	o.Settings.OsdText = ""

	o.Elements.PrivacyMode = new(Child)
	o.Elements.PrivacyMode.Value = false
	o.Elements.PrivacyMode.run = o.setPrivacy

	o.Elements.NightModeAuto = new(Child)
	o.Elements.NightModeAuto.Value = true
	o.Elements.NightModeAuto.run = o.setNightModeAuto

	o.Elements.NightMode = new(Child)
	o.Elements.NightMode.Value = true
	o.Elements.NightMode.run = o.setNightMode

	o.Elements.Indicator = new(Child)
	o.Elements.Indicator.Value = true
	o.Elements.Indicator.run = o.setLed

	o.Elements.AutotrackingMode = new(Child)
	o.Elements.AutotrackingMode.Value = false
	o.Elements.AutotrackingMode.run = o.setAutotracking

	o.Settings.PresetChangeOsd = new(Child)
	o.Settings.PresetChangeOsd.Value = false
	o.Settings.PresetChangeOsd.run = fnil

	o.Elements.DetectMode = new(Child)
	o.Elements.DetectMode.Value = false
	o.Elements.DetectMode.run = o.setDetect

	o.Settings.DetectSensitivity = 1

	o.Settings.DetectSoundAlternativeMode = new(Child)
	o.Settings.DetectSoundAlternativeMode.Value = false
	o.Settings.DetectSoundAlternativeMode.run = o.setDetect

	o.Settings.DetectEnableSound = new(Child)
	o.Settings.DetectEnableSound.Value = true
	o.Settings.DetectEnableSound.run = o.setDetect

	o.Settings.DetectEnableFlash = new(Child)
	o.Settings.DetectEnableFlash.Value = false
	o.Settings.DetectEnableFlash.run = o.setDetect

	o.Elements.AlarmMode = new(Child)
	o.Elements.AlarmMode.Value = false
	o.Elements.AlarmMode.run = o.setAlarm

	o.Settings.Time = new(Child)
	o.Settings.Time.Value = true
	o.Settings.Time.run = o.getTime

	o.Settings.User = new(Child)
	o.Settings.User.Value = true
	o.Settings.User.run = o.getUser

	o.NextPreset = o.setNextPreset
	o.Reboot = o.rebootDevice
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
	o.stokId = result.Result.Stok
	o.userGroup = result.Result.UserGroup
	o.hostURL += `/stok=` + o.stokId + `/ds`
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
	o.deviceId = result.DeviceInfo.BasicInfo.DevID
}

// Manual move
//  10 = 10 degree
// -10 = 10 degree reverse
func (o *Tapo) setMovePosition(x int, y int) {
	o.update()
	bodyStruct := new(MovePosition)
	bodyStruct.Method = MethodDo
	bodyStruct.Motor.Move.XCoord = strconv.Itoa(x)
	bodyStruct.Motor.Move.YCoord = strconv.Itoa(y)
	data, _ := json.Marshal(bodyStruct)
	o.query(data)
}

// Get all making Presets in App
func (o *Tapo) getPresets() {
	o.update()
	bodyStruct := new(PresetList)
	result := new(PresetListReturn)
	bodyStruct.Method = MethodGet
	bodyStruct.Preset.Name = []string{"preset"}
	data, _ := json.Marshal(bodyStruct)
	json.Unmarshal(o.query(data), &result)
	for i, v := range result.Preset.Preset.ID {
		pos := new(presets)
		pos.Id = v
		pos.Name = result.Preset.Preset.Name[i]
		o.presets = append(o.presets, pos)

	}
}

// Switch to next preset
func (o *Tapo) setNextPreset() {
	o.update()
	o.lastPosition = o.rLast()
	max := new(presets)
	max.Id = "-9999"
	min := new(presets)
	min.Id = "9999"
	next := new(presets)
	next.Id = "0"
	check := true
	for _, v := range o.presets {
		vId, _ := strconv.Atoi(v.Id)
		minId, _ := strconv.Atoi(min.Id)
		maxId, _ := strconv.Atoi(max.Id)
		if vId < minId {
			min = v
		}
		if vId > maxId {
			max = v
		}
		if vId > o.lastPosition && check {
			next = v
			check = false
		}
	}
	if next.Id == "0" {
		next = min
		o.wLast(min.Id)
	}
	bodyStruct := new(NextPreset)
	bodyStruct.Method = MethodDo
	bodyStruct.Preset.GotoPreset.ID = next.Id
	data, _ := json.Marshal(bodyStruct)
	if o.Settings.PresetChangeOsd.Value {
		o.Settings.OsdText = next.Name
		o.Settings.VisibleOsdText.Value = true
		o.Settings.VisibleOsdTime.Value = true
		o.setOsd()
	}
	o.query(data)
	o.wLast(next.Id)
}

// Write log last file
func (o *Tapo) wLast(text string) {
	ioutil.WriteFile(o.LastFile+`/`+LastFileName, []byte(text), 0775)
}

// Read log last file
func (o *Tapo) rLast() int {
	if last_, err := ioutil.ReadFile(o.LastFile + `/` + LastFileName); err == nil {
		last, _ := strconv.Atoi(string(last_))
		return last
	} else {
		os.Create(o.LastFile + `/` + LastFileName)
	}
	return 0
}

// Run all presets with timer beetween
func (o *Tapo) runAllPresets(timer string) {
	dur_, _ := time.ParseDuration(timer)
	for range o.presets {
		time.Sleep(dur_)
		o.setNextPreset()
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

// Take value
func sBool(val bool) string {
	if val {
		return "on"
	}
	return "off"
}

// Take value
func bsBool(val bool) string {
	if val {
		return "1"
	}
	return "0"
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
	bodyStruct.MsgAlarm.Chn1MsgAlarmInfo.AlarmType = bsBool(o.Settings.DetectSoundAlternativeMode.Value)
	bodyStruct.MsgAlarm.Chn1MsgAlarmInfo.LightType = "1"
	bodyStruct.MsgAlarm.Chn1MsgAlarmInfo.Enabled = sBool(o.Elements.AlarmMode.Value)
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
	bodyStruct.Led.Config.Enabled = sBool(o.Elements.Indicator.Value)
	data, _ := json.Marshal(bodyStruct)
	o.query(data)

}

// Get Time
func (o *Tapo) getTime() {
	o.update()
	result := new(getTimeRet)
	bodyStruct := new(getTime)
	bodyStruct.Method = MethodGet
	bodyStruct.System.Name = []string{"clock_status"}
	data, _ := json.Marshal(bodyStruct)
	json.Unmarshal(o.query(data), &result)
	o.TimeStr = result.System.ClockStatus.LocalTime
	p(string(data))
	p(string(o.query(data)))
}

// Get User
func (o *Tapo) getUser() {
	o.update()
	//result := new(getTimeRet)
	bodyStruct := new(getUser)
	bodyStruct.Method = MethodGet
	//	bodyStruct.System.GetUserID = "null"
	data, _ := json.Marshal(bodyStruct)
	//json.Unmarshal(o.query(data), &result)
	//o.TimeStr = result.System.ClockStatus.LocalTime
	p(string(data))
	p(string(o.query(data)))
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
	bodyStruct.MotionDetection.MotionDet.Enabled = sBool(o.Elements.DetectMode.Value)
	data, _ := json.Marshal(bodyStruct)
	o.query(data)
}

// Turn camera in private mode with stop video channel
func (o *Tapo) setPrivacy() {
	o.update()
	bodyStruct := new(privacy)
	bodyStruct.Method = MethodSet
	bodyStruct.LensMask.LensMaskInfo.Enabled = sBool(o.Elements.PrivacyMode.Value)
	data, _ := json.Marshal(bodyStruct)
	o.query(data)
}

// Turn irc flashlight
func (o *Tapo) setNightMode() {
	o.update()
	bodyStruct := new(nightMode)
	bodyStruct.Method = MethodSet
	bodyStruct.Image.Common.InfType = sBool(o.Elements.NightMode.Value)
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
	bodyStruct.TargetTrack.TargetTrackInfo.Enabled = sBool(o.Elements.AutotrackingMode.Value)
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
	bodyStruct.OSD.Date.Enabled = sBool(o.Settings.VisibleOsdTime.Value)
	bodyStruct.OSD.Date.XCoor = 0
	bodyStruct.OSD.Date.YCoor = 0
	bodyStruct.OSD.Font.Color = "white"
	bodyStruct.OSD.Font.ColorType = "auto"
	bodyStruct.OSD.Font.Display = "ntnb"
	bodyStruct.OSD.Font.Size = "auto"
	bodyStruct.OSD.LabelInfo1.Enabled = sBool(o.Settings.VisibleOsdText.Value)
	bodyStruct.OSD.LabelInfo1.Text = o.Settings.OsdText
	bodyStruct.OSD.LabelInfo1.XCoor = 0
	bodyStruct.OSD.LabelInfo1.YCoor = 450
	//---china weeks---
	bodyStruct.OSD.Week.Enabled = sBool(false)
	bodyStruct.OSD.Week.XCoor = 0
	bodyStruct.OSD.Week.YCoor = 0
	//---china weeks---
	data, _ := json.Marshal(bodyStruct)
	o.query(data)
}

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

func (o *Child) On() {
	o.Value = true
	o.run()
}

func (o *Child) Off() {
	o.Value = false
	o.run()
}

func (o *Tapo) On(s Action) {
	s.On()
}

func (o *Tapo) Off(s Action) {
	s.Off()
}

func (o *Tapo) MoveRight(val int) {
	o.setMovePosition(val, 0)
	time.Sleep(5 * time.Second)
}

func (o *Tapo) MoveLeft(val int) {
	o.setMovePosition(-val, 0)
	time.Sleep(5 * time.Second)
}

func (o *Tapo) MoveUp(val int) {
	o.setMovePosition(0, val)
	time.Sleep(5 * time.Second)
}

func (o *Tapo) MoveDown(val int) {
	o.setMovePosition(0, -val)
	time.Sleep(5 * time.Second)
}

func (o *Tapo) MoveTest() {
	o.runAllPresets("10s")
}
