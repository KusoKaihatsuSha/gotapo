// Package gotapo working
// with camera tapo like c200, c310
// by http
package gotapo

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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

	EncryptType = "3"
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
	NightMode            *child
	NightModeAuto        *child
	PrivacyMode          *child
	Indicator            *child
	DetectMode           *child
	AutotrackingMode     *child
	AlarmMode            *child
	ImageCorrection      *child
	ImageFlip            *child
	MoveX                string
	MoveY                string
	AlarmModeUpdateFlash *child
	AlarmModeUpdateSound *child
	DetectPersonMode     *child
	DetectModeUpdateSens *child
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
	Move                       *child
}

// child assignment of function
type child struct {
	Value bool
	run   func()
}

// Tapo is general type with Vals
type Tapo struct {
	Parameters           map[string]string
	Host                 string
	Port                 string
	UserDef              string
	User                 string
	Password             string
	UserID               string
	Rotate               bool
	FishEye              bool
	Flip                 bool
	stokID               string
	TimeStr              string
	userGroup            string
	hashedPassword       string
	hashedPasswordMD5    string
	hashedPasswordSha256 string
	hostURL              string
	hostURLStok          string
	deviceModel          string
	deviceID             string
	presets              []*presets
	lastPosition         int
	LastFile             string
	Elements             *elements
	Settings             *settings
	NextPreset           func()
	Reboot               func()
	InsecureAuth         bool
	Iv                   []byte
	Key                  []byte
	Seq                  string
	Encrypt              bool
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
		StartSeq  *int   `json:"start_seq"`
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
	Result struct {
		Responses []struct {
			Method string `json:"method"`
			Result struct {
				DeviceInfo struct {
					BasicInfo struct {
						Ffs         bool   `json:"ffs"`
						DeviceType  string `json:"device_type"`
						DeviceModel string `json:"device_model"`
						DeviceName  string `json:"device_name"`
						DeviceInfo  string `json:"device_info"`
						HwVersion   string `json:"hw_version"`
						SwVersion   string `json:"sw_version"`
						DeviceAlias string `json:"device_alias"`
						Features    string `json:"features"`
						Barcode     string `json:"barcode"`
						Mac         string `json:"mac"`
						DevID       string `json:"dev_id"`
						OemID       string `json:"oem_id"`
						HwDesc      string `json:"hw_desc"`
					} `json:"basic_info"`
				} `json:"device_info"`
			} `json:"result"`
			ErrorCode int `json:"error_code"`
		} `json:"responses"`
	} `json:"result"`
	ErrorCode int `json:"error_code"`
}

type moveTo struct {
	Method string `json:"method"`
	Motor  struct {
		MoveStep struct {
			Derection string `json:"direction"`
		} `json:"movestep"`
	} `json:"motor"`
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
	Result struct {
		Responses []struct {
			Method string `json:"method"`
			Result struct {
				Preset struct {
					Preset struct {
						ID           []string `json:"id"`
						Name         []string `json:"name"`
						PositionPan  []string `json:"position_pan"`
						PositionTilt []string `json:"position_tilt"`
						ReadOnly     []string `json:"read_only"`
					} `json:"preset"`
				} `json:"preset"`
			} `json:"result"`
			ErrorCode int `json:"error_code"`
		} `json:"responses"`
	} `json:"result"`
	ErrorCode int `json:"error_code"`
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
	Result struct {
		Responses []struct {
			Method string `json:"method"`
			Result struct {
				Image struct {
					Switch struct {
						Name              string `json:".name"`
						Type              string `json:".type"`
						SwitchMode        string `json:"switch_mode"`
						ScheduleStartTime string `json:"schedule_start_time"`
						ScheduleEndTime   string `json:"schedule_end_time"`
						FlipType          string `json:"flip_type"`
						RotateType        string `json:"rotate_type"`
						Ldc               string `json:"ldc"`
						NightVisionMode   string `json:"night_vision_mode"`
						WtlIntensityLevel string `json:"wtl_intensity_level"`
					} `json:"switch"`
					Common struct {
						Name                  string `json:".name"`
						Type                  string `json:".type"`
						Luma                  string `json:"luma"`
						Contrast              string `json:"contrast"`
						Chroma                string `json:"chroma"`
						Saturation            string `json:"saturation"`
						Sharpness             string `json:"sharpness"`
						ExpType               string `json:"exp_type"`
						Shutter               string `json:"shutter"`
						FocusType             string `json:"focus_type"`
						FocusLimited          string `json:"focus_limited"`
						ExpGain               string `json:"exp_gain"`
						InfStartTime          string `json:"inf_start_time"`
						InfEndTime            string `json:"inf_end_time"`
						InfSensitivity        string `json:"inf_sensitivity"`
						InfDelay              string `json:"inf_delay"`
						WideDynamic           string `json:"wide_dynamic"`
						LightFreqMode         string `json:"light_freq_mode"`
						WdGain                string `json:"wd_gain"`
						WbType                string `json:"wb_type"`
						WbRGain               string `json:"wb_R_gain"`
						WbGGain               string `json:"wb_G_gain"`
						WbBGain               string `json:"wb_B_gain"`
						LockRedGain           string `json:"lock_red_gain"`
						LockGrGain            string `json:"lock_gr_gain"`
						LockGbGain            string `json:"lock_gb_gain"`
						LockBlueGain          string `json:"lock_blue_gain"`
						LockRedColton         string `json:"lock_red_colton"`
						LockGreenColton       string `json:"lock_green_colton"`
						LockBlueColton        string `json:"lock_blue_colton"`
						LockSource            string `json:"lock_source"`
						AreaCompensation      string `json:"area_compensation"`
						Smartir               string `json:"smartir"`
						SmartirLevel          string `json:"smartir_level"`
						HighLightCompensation string `json:"high_light_compensation"`
						Dehaze                string `json:"dehaze"`
						InfType               string `json:"inf_type"`
					} `json:"common"`
				} `json:"image"`
			} `json:"result"`
			ErrorCode int `json:"error_code"`
		} `json:"responses"`
	} `json:"result"`
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
type getOSD struct {
	Method string `json:"method"`
	Data   struct {
		Name  []string `json:"name"`
		Table []string `json:"table"`
	} `json:"OSD"`
}

type setLed struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Value string `json:"enabled"`
		} `json:"config"`
	} `json:"led"`
}

type setPersonDetect struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Value string `json:"enabled"`
		} `json:"detection"`
	} `json:"people_detection"`
}

type deviceInfo struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name []string `json:"name"`
		} `json:"device_info"`
	} `json:"params"`
}
type detectionConfig struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name []string `json:"name"`
		} `json:"motion_detection"`
	} `json:"params"`
}
type detectionConfigResponse struct {
	Result struct {
		Responses []struct {
			Method string `json:"method"`
			Result struct {
				MotionDetection struct {
					MotionDet struct {
						Name               string `json:".name"`
						Type               string `json:".type"`
						Sensitivity        string `json:"sensitivity"`
						DigitalSensitivity string `json:"digital_sensitivity"`
						Enabled            string `json:"enabled"`
					} `json:"motion_det"`
				} `json:"motion_detection"`
			} `json:"result"`
			ErrorCode int `json:"error_code"`
		} `json:"responses"`
	} `json:"result"`
	ErrorCode int `json:"error_code"`
}
type personDetectionConfig struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name []string `json:"name"`
		} `json:"people_detection"`
	} `json:"params"`
}
type vehicleDetectionConfig struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name []string `json:"name"`
		} `json:"vehicle_detection"`
	} `json:"params"`
}
type bcdConfig struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name []string `json:"name"`
		} `json:"sound_detection"`
	} `json:"params"`
}
type petDetectionConfig struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name []string `json:"name"`
		} `json:"pet_detection"`
	} `json:"params"`
}
type barkDetectionConfig struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name []string `json:"name"`
		} `json:"bark_detection"`
	} `json:"params"`
}
type meowDetectionConfig struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name []string `json:"name"`
		} `json:"meow_detection"`
	} `json:"params"`
}
type glassDetectionConfig struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name []string `json:"name"`
		} `json:"glass_detection"`
	} `json:"params"`
}
type tamperDetectionConfig struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name string `json:"name"`
		} `json:"tamper_detection"`
	} `json:"params"`
}
type lensMaskConfig struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name []string `json:"name"`
		} `json:"lens_mask"`
	} `json:"params"`
}
type ldc struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name []string `json:"name"`
		} `json:"image"`
	} `json:"params"`
}
type lastAlarmInfo struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name []string `json:"name"`
		} `json:"msg_alarm"`
	} `json:"params"`
}

type lastAlarmInfoResponse struct {
	Result struct {
		Responses []struct {
			Method string `json:"method"`
			Result struct {
				MsgAlarm struct {
					Chn1MsgAlarmInfo struct {
						Name              string   `json:".name"`
						Type              string   `json:".type"`
						SoundAlarmEnabled string   `json:"sound_alarm_enabled"`
						LightAlarmEnabled string   `json:"light_alarm_enabled"`
						AlarmType         string   `json:"alarm_type"`
						LightType         string   `json:"light_type"`
						AlarmMode         []string `json:"alarm_mode"`
						Enabled           string   `json:"enabled"`
					} `json:"chn1_msg_alarm_info"`
				} `json:"msg_alarm"`
			} `json:"result"`
			ErrorCode int `json:"error_code"`
		} `json:"responses"`
	} `json:"result"`
	ErrorCode int `json:"error_code"`
}

type ledStatus struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name []string `json:"name"`
		} `json:"led"`
	} `json:"params"`
}
type targetTrackConfig struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name []string `json:"name"`
		} `json:"target_track"`
	} `json:"params"`
}
type presetConfig struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name []string `json:"name"`
		} `json:"preset"`
	} `json:"params"`
}
type firmwareUpdateStatus struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name []string `json:"name"`
		} `json:"cloud_config"`
	} `json:"params"`
}
type mediaEncrypt struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name []string `json:"name"`
		} `json:"cet"`
	} `json:"params"`
}
type connectionType struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name []string `json:"get_connection_type"`
		} `json:"network"`
	} `json:"params"`
}
type alarmConfig struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
		} `json:"msg_alarm"`
	} `json:"params"`
}
type alarmPlan struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
		} `json:"msg_alarm_plan"`
	} `json:"params"`
}
type sirenTypeList struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
		} `json:"msg_alarm"`
	} `json:"params"`
}
type lightTypeList struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
		} `json:"msg_alarm"`
	} `json:"params"`
}
type sirenStatus struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
		} `json:"msg_alarm"`
	} `json:"params"`
}
type lightFrequencyInfo struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name string `json:"name"`
		} `json:"image"`
	} `json:"params"`
}
type lightFrequencyCapability struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name string `json:"name"`
		} `json:"image"`
	} `json:"params"`
}
type childDeviceList struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			StartIndex int `json:"start_index"`
		} `json:"childControl"`
	} `json:"params"`
}
type rotationStatus struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name []string `json:"name"`
		} `json:"image"`
	} `json:"params"`
}
type nightVisionModeConfig struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name string `json:"name"`
		} `json:"image"`
	} `json:"params"`
}
type whitelampStatus struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			GetWtlStatus []string `json:"get_wtl_status"`
		} `json:"image"`
	} `json:"params"`
}
type whitelampConfig struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name string `json:"name"`
		} `json:"image"`
	} `json:"params"`
}
type msgPushConfig struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name []string `json:"name"`
		} `json:"msg_push"`
	} `json:"params"`
}
type sdCardStatus struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Table []string `json:"table"`
		} `json:"harddisk_manage"`
	} `json:"params"`
}
type circularRecordingConfig struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name string `json:"name"`
		} `json:"harddisk_manage"`
	} `json:"params"`
}
type recordPlan struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name []string `json:"name"`
		} `json:"record_plan"`
	} `json:"params"`
}
type firmwareAutoUpgradeConfig struct {
	Method string `json:"method"`
	Data   struct {
		Type struct {
			Name []string `json:"name"`
		} `json:"auto_upgrade"`
	} `json:"params"`
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

type queryResponse struct {
	ErrorCode int `json:"error_code"`
	Seq       int `json:"seq"`
	Result    struct {
		Response string `json:"response"`
	} `json:"result"`
}

type many struct {
	Method string `json:"method"`
	Params struct {
		Requests []any `json:"requests"`
	} `json:"params"`
}

type secure struct {
	Method string `json:"method"`
	Params struct {
		Request string `json:"request"`
	} `json:"params"`
}

// Auth with nonce usage
type loginInsecure struct {
	Method string `json:"method"`
	Params struct {
		Cnonce       string `json:"cnonce"`
		EncryptType  string `json:"encrypt_type"`
		DigestPasswd string `json:"digest_passwd,omitempty"`
		Username     string `json:"username"`
	} `json:"params"`
}

// Auth with nonce usage
type loginInsecureResponse struct {
	ErrorCode int `json:"error_code"`
	Result    struct {
		Data struct {
			Code          int      `json:"code"`
			EncryptType   []string `json:"encrypt_type"`
			Key           string   `json:"key"`
			Nonce         string   `json:"nonce"`
			DeviceConfirm string   `json:"device_confirm"`
		} `json:"data"`
	} `json:"result"`
}

// nil func
func fnil() {
}

func secureTemplate(values ...any) secure {
	t := secure{}
	t.Method = "securePassthrough"
	t.Params.Request = values[0].(string)
	return t
}

func manyTemplate(values ...any) many {
	t := many{}
	t.Method = "multipleRequest"
	t.Params.Requests = append(t.Params.Requests, values...)
	return t
}

func osdTemplate(values ...any) osd {
	t := osd{}
	t.Method = MethodSet
	t.OSD.Date.Enabled = values[0].(string)
	t.OSD.Date.XCoor = 0
	t.OSD.Date.YCoor = 0
	t.OSD.Font.Color = "white"
	t.OSD.Font.ColorType = "auto"
	t.OSD.Font.Display = "ntnb"
	t.OSD.Font.Size = "auto"
	t.OSD.LabelInfo1.Enabled = values[1].(string)
	t.OSD.LabelInfo1.Text = values[2].(string)
	t.OSD.LabelInfo1.XCoor = 0
	t.OSD.LabelInfo1.YCoor = 450
	//---china weeks---
	t.OSD.Week.Enabled = new(Types).xBool(false).Default
	t.OSD.Week.XCoor = 0
	t.OSD.Week.YCoor = 0
	return t
}

func alarmTemplate(values ...any) alarm {
	t := alarm{}
	t.Method = MethodSet
	t.MsgAlarm.Chn1MsgAlarmInfo.AlarmType = values[0].(string)
	t.MsgAlarm.Chn1MsgAlarmInfo.LightType = "1"
	t.MsgAlarm.Chn1MsgAlarmInfo.AlarmMode = values[1].([]string)
	t.MsgAlarm.Chn1MsgAlarmInfo.Enabled = values[2].(string)
	return t
}

func nextPresetTemplate(values ...any) nextPreset {
	t := nextPreset{}
	t.Method = MethodDo
	t.Preset.GotoPreset.ID = values[0].(string)
	return t
}

func loginNewTemplate(values ...any) loginInsecure {
	t := loginInsecure{}
	t.Method = MethodLogin
	t.Params.Username = values[0].(string)
	t.Params.EncryptType = EncryptType
	t.Params.DigestPasswd = values[1].(string)
	return t
}

func updateStokTemplate(values ...any) updateStok {
	t := updateStok{}
	t.Method = MethodLogin
	t.Params.Hashed = true
	t.Params.Username = values[0].(string)
	t.Params.Password = values[1].(string)
	return t
}

func getTimeTemplate(values ...any) getTime {
	t := getTime{}
	t.Method = MethodGet
	t.System.Name = []string{"clock_status"}
	return t
}

func setImageCorrectionTemplate(values ...any) setImageCorrection {
	t := setImageCorrection{}
	t.Method = MethodSet
	t.Image.Switch.Ldc = values[0].(string)
	return t
}

func setImageFlipTemplate(values ...any) setImageFlip {
	t := setImageFlip{}
	t.Method = MethodSet
	t.Image.Switch.FlipType = values[0].(string)
	return t
}

func detectTemplate(values ...any) detect {
	t := detect{}
	t.Method = MethodSet
	switch values[1].(int) {
	case 1:
		t.MotionDetection.MotionDet.DigitalSensitivity = "20"
	case 2:
		t.MotionDetection.MotionDet.DigitalSensitivity = "50"
	case 3:
		t.MotionDetection.MotionDet.DigitalSensitivity = "80"
	default:
		t.MotionDetection.MotionDet.DigitalSensitivity = "20"
	}
	t.MotionDetection.MotionDet.Enabled = values[0].(string)
	return t
}

func privacyTemplate(values ...any) privacy {
	t := privacy{}
	t.Method = MethodSet
	t.LensMask.LensMaskInfo.Enabled = values[0].(string)
	return t
}

func nightModeTemplate(values ...any) nightMode {
	t := nightMode{}
	t.Method = MethodSet
	t.Image.Common.InfType = values[0].(string)
	return t
}

func autotrackingTemplate(values ...any) autotracking {
	t := autotracking{}
	t.Method = MethodSet
	t.TargetTrack.TargetTrackInfo.Enabled = values[0].(string)
	return t
}

func getOSDTemplate(values ...any) getOSD {
	t := getOSD{}
	t.Method = MethodGet
	t.Data.Name = []string{"date", "week", "font"}
	t.Data.Table = []string{"label_info"}
	return t
}

func loginInitTemplate(values ...any) loginInsecure {
	t := loginInsecure{}
	t.Method = MethodLogin
	t.Params.Username = values[0].(string)
	t.Params.EncryptType = EncryptType
	return t
}

func rebootTemplate(values ...any) reboot {
	t := reboot{}
	t.Method = MethodDo
	t.System.Reboot = "null"
	return t
}

func moveToTemplate(values ...any) moveTo {
	t := moveTo{}
	t.Method = MethodDo
	t.Motor.MoveStep.Derection = values[0].(string)
	return t
}

func movePositionTemplate(values ...any) movePosition {
	t := movePosition{}
	t.Method = MethodDo
	t.Motor.Move.XCoord = strconv.Itoa(values[0].(int))
	t.Motor.Move.YCoord = strconv.Itoa(values[1].(int))
	return t
}

func setLedTemplate(values ...any) setLed {
	t := setLed{}
	t.Method = MethodSet
	t.Data.Type.Value = values[0].(string)
	return t
}

func setPersonDetectTemplate(values ...any) setPersonDetect {
	t := setPersonDetect{}
	t.Method = MethodSet
	t.Data.Type.Value = values[0].(string)
	return t
}

func deviceInfoTemplate(values ...any) deviceInfo {
	t := deviceInfo{}
	t.Method = "getDeviceInfo"
	t.Data.Type.Name = []string{"basic_info"}
	return t
}

func detectionConfigTemplate(values ...any) detectionConfig {
	t := detectionConfig{}
	t.Method = "getDetectionConfig"
	t.Data.Type.Name = []string{"motion_det"}
	return t
}

func personDetectionConfigTemplate(values ...any) personDetectionConfig {
	t := personDetectionConfig{}
	t.Method = "getPersonDetectionConfig"
	t.Data.Type.Name = []string{"detection"}
	return t
}

func vehicleDetectionConfigTemplate(values ...any) vehicleDetectionConfig {
	t := vehicleDetectionConfig{}
	t.Method = "getVehicleDetectionConfig"
	t.Data.Type.Name = []string{"detection"}
	return t
}

func bcdConfigTemplate(values ...any) bcdConfig {
	t := bcdConfig{}
	t.Method = "getBCDConfig"
	t.Data.Type.Name = []string{"bcd"}
	return t
}

func petDetectionConfigTemplate(values ...any) petDetectionConfig {
	t := petDetectionConfig{}
	t.Method = "getPetDetectionConfig"
	t.Data.Type.Name = []string{"detection"}
	return t
}

func barkDetectionConfigTemplate(values ...any) barkDetectionConfig {
	t := barkDetectionConfig{}
	t.Method = "getBarkDetectionConfig"
	t.Data.Type.Name = []string{"detection"}
	return t
}

func meowDetectionConfigTemplate(values ...any) meowDetectionConfig {
	t := meowDetectionConfig{}
	t.Method = "getMeowDetectionConfig"
	t.Data.Type.Name = []string{"detection"}
	return t
}

func glassDetectionConfigTemplate(values ...any) glassDetectionConfig {
	t := glassDetectionConfig{}
	t.Method = "getGlassDetectionConfig"
	t.Data.Type.Name = []string{"detection"}
	return t
}

func tamperDetectionConfigTemplate(values ...any) tamperDetectionConfig {
	t := tamperDetectionConfig{}
	t.Method = "getTamperDetectionConfig"
	t.Data.Type.Name = "tamper_det"
	return t
}

func lensMaskConfigTemplate(values ...any) lensMaskConfig {
	t := lensMaskConfig{}
	t.Method = "getLensMaskConfig"
	t.Data.Type.Name = []string{"lens_mask_info"}
	return t
}

func ldcTemplate(values ...any) ldc {
	t := ldc{}
	t.Method = "getLdc"
	t.Data.Type.Name = []string{"switch", "common"}
	return t
}

func lastAlarmInfoTemplate(values ...any) lastAlarmInfo {
	t := lastAlarmInfo{}
	t.Method = "getLastAlarmInfo"
	t.Data.Type.Name = []string{"chn1_msg_alarm_info"}
	return t
}

func ledStatusTemplate(values ...any) ledStatus {
	t := ledStatus{}
	t.Method = "getLedStatus"
	t.Data.Type.Name = []string{"config"}
	return t
}

func targetTrackConfigTemplate(values ...any) targetTrackConfig {
	t := targetTrackConfig{}
	t.Method = "getTargetTrackConfig"
	t.Data.Type.Name = []string{"target_track_info"}
	return t
}

func presetConfigTemplate(values ...any) presetConfig {
	t := presetConfig{}
	t.Method = "getPresetConfig"
	t.Data.Type.Name = []string{"preset"}
	return t
}

func firmwareUpdateStatusTemplate(values ...any) firmwareUpdateStatus {
	t := firmwareUpdateStatus{}
	t.Method = "getFirmwareUpdateStatus"
	t.Data.Type.Name = []string{"upgrade_status"}
	return t
}

func mediaEncryptTemplate(values ...any) mediaEncrypt {
	t := mediaEncrypt{}
	t.Method = "getMediaEncrypt"
	t.Data.Type.Name = []string{"media_encrypt"}
	return t
}

func connectionTypeTemplate(values ...any) connectionType {
	t := connectionType{}
	t.Method = "getConnectionType"
	t.Data.Type.Name = []string{"get_connection_type"}
	return t
}

func lightFrequencyInfoTemplate(values ...any) lightFrequencyInfo {
	t := lightFrequencyInfo{}
	t.Method = "getLightFrequencyInfo"
	t.Data.Type.Name = "common"
	return t
}

func lightFrequencyCapabilityTemplate(values ...any) lightFrequencyCapability {
	t := lightFrequencyCapability{}
	t.Method = "getLightFrequencyCapability"
	t.Data.Type.Name = "common"
	return t
}

func childDeviceListTemplate(values ...any) childDeviceList {
	t := childDeviceList{}
	t.Method = "getChildDeviceList"
	t.Data.Type.StartIndex = 0
	return t
}

func rotationStatusTemplate(values ...any) rotationStatus {
	t := rotationStatus{}
	t.Method = "getRotationStatus"
	t.Data.Type.Name = []string{"switch"}
	return t
}

func nightVisionModeConfigTemplate(values ...any) nightVisionModeConfig {
	t := nightVisionModeConfig{}
	t.Method = "getNightVisionModeConfig"
	t.Data.Type.Name = "switch"
	return t
}

func whitelampStatusTemplate(values ...any) whitelampStatus {
	t := whitelampStatus{}
	t.Method = "getWhitelampStatus"
	t.Data.Type.GetWtlStatus = []string{"null"}
	return t
}

func whitelampConfigTemplate(values ...any) whitelampConfig {
	t := whitelampConfig{}
	t.Method = "getWhitelampConfig"
	t.Data.Type.Name = "switch"
	return t
}

func msgPushConfigTemplate(values ...any) msgPushConfig {
	t := msgPushConfig{}
	t.Method = "getMsgPushConfig"
	t.Data.Type.Name = []string{"chn1_msg_push_info"}
	return t
}

func sdCardStatusTemplate(values ...any) sdCardStatus {
	t := sdCardStatus{}
	t.Method = "getSdCardStatus"
	t.Data.Type.Table = []string{"hd_info"}
	return t
}

func circularRecordingConfigTemplate(values ...any) circularRecordingConfig {
	t := circularRecordingConfig{}
	t.Method = "getCircularRecordingConfig"
	t.Data.Type.Name = "harddisk"
	return t
}

func recordPlanTemplate(values ...any) recordPlan {
	t := recordPlan{}
	t.Method = "getRecordPlan"
	t.Data.Type.Name = []string{"chn1_channel"}
	return t
}

func firmwareAutoUpgradeConfigTemplate(values ...any) firmwareAutoUpgradeConfig {
	t := firmwareAutoUpgradeConfig{}
	t.Method = "getFirmwareAutoUpgradeConfig"
	t.Data.Type.Name = []string{"common"}
	return t
}

// Connect is general function for connecting to Camera
func Connect(host string, user string, password string) *Tapo {
	o := new(Tapo)
	o.LastFile, _ = os.Getwd()
	o.Host = host
	o.Port = "443"
	o.User = user
	o.Password = password
	o.init()
	o.auth()
	o.getDevice()
	o.getImageSettings()
	o.getPresets()

	return o
}

// Firsty initialise
func (o *Tapo) init() {
	o.hashedPasswordMD5 = hashNHexOld(o.Password)
	o.hashedPasswordSha256 = hashNHex(o.Password)
	o.hostURL = `https://` + o.Host + `:` + o.Port

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
	o.Settings.VisibleOsdTime.run = o.setOsdTime

	o.Settings.VisibleOsdText = new(child)
	o.Settings.VisibleOsdText.Value = false
	o.Settings.VisibleOsdText.run = o.setOsdText

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

	o.Elements.DetectModeUpdateSens = new(child)
	o.Elements.DetectModeUpdateSens.Value = false
	o.Elements.DetectModeUpdateSens.run = o.updateSens

	o.Elements.DetectMode = new(child)
	o.Elements.DetectMode.Value = false
	o.Elements.DetectMode.run = o.setDetect

	o.Elements.DetectPersonMode = new(child)
	o.Elements.DetectPersonMode.Value = false
	o.Elements.DetectPersonMode.run = o.setDetectPerson

	o.Settings.DetectSensitivity = 1

	o.Settings.DetectSoundAlternativeMode = new(child)
	o.Settings.DetectSoundAlternativeMode.Value = false
	o.Settings.DetectSoundAlternativeMode.run = fnil

	o.Settings.DetectEnableSound = new(child)
	o.Settings.DetectEnableSound.Value = true
	o.Settings.DetectEnableSound.run = fnil

	o.Settings.DetectEnableFlash = new(child)
	o.Settings.DetectEnableFlash.Value = false
	o.Settings.DetectEnableFlash.run = fnil

	o.Elements.AlarmMode = new(child)
	o.Elements.AlarmMode.Value = false
	o.Elements.AlarmMode.run = o.setAlarm

	o.Elements.AlarmModeUpdateFlash = new(child)
	o.Elements.AlarmModeUpdateFlash.Value = false
	o.Elements.AlarmModeUpdateFlash.run = o.updateAlarmFlash

	o.Elements.AlarmModeUpdateSound = new(child)
	o.Elements.AlarmModeUpdateSound.Value = false
	o.Elements.AlarmModeUpdateSound.run = o.updateAlarmSound

	o.Settings.Time = new(child)
	o.Settings.Time.Value = true
	o.Settings.Time.run = o.getTime

	o.Settings.PrintImageSettings = new(child)
	o.Settings.PrintImageSettings.Value = true
	o.Settings.PrintImageSettings.run = o.getImageSettings

	o.Elements.ImageCorrection = new(child)
	o.Elements.ImageCorrection.Value = true
	o.Elements.ImageCorrection.run = o.setImageCorrection

	o.Elements.ImageFlip = new(child)
	o.Elements.ImageFlip.Value = true
	o.Elements.ImageFlip.run = o.setImageFlip

	o.Elements.MoveX = ""
	o.Elements.MoveY = ""
	o.Settings.Move = new(child)
	o.Settings.Move.Value = true
	o.Settings.Move.run = o.setMoveAction

	o.NextPreset = o.setNextPreset
	o.Reboot = o.rebootDevice
}

// Pack and encode request
func pack(request any, key, iv []byte) secure {
	requestJSON, err := json.Marshal(request)
	if err != nil {
		p("Error pack query")
	}
	return secureTemplate(
		encodeB64(
			encodeAES(requestJSON, key, iv),
		),
	)
	//return template[secure](encodeB64(encodeAES(requestJSON, key, iv)))
}

// POST query to cam
func (o *Tapo) query(data any, host string, encrypt bool) []byte {
	if encrypt {
		data = pack(data, o.Key, o.Iv)
	}
	dataBody, _ := json.Marshal(data)
	body := bytes.NewReader(dataBody)
	req, _ := http.NewRequest("POST", host, body)
	for k, v := range o.Parameters {
		req.Header.Add(k, v)
	}
	if encrypt {
		req.Header.Add("Seq", o.Seq)
		req.Header.Add("Tapo_tag", hashNHex(hashNHex(o.hashedPassword)+string(dataBody)+o.Seq))
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
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte{}
	}
	if encrypt {
		result := new(queryResponse)
		json.Unmarshal(b, &result)
		return []byte(decodeAES([]byte(decodeB64(result.Result.Response)), o.Key, o.Iv))
	}
	defer resp.Body.Close()
	return b
}

// Check insecure of authorise.
// At this moment the simplest way is send hash(sha256)
// (but not best and stable).
// On firmware >= 1.3.9(11 for new hardware) old type of authorise with hash(md5) will not valid
func (o *Tapo) auth() {
	o.hashedPassword = o.hashedPasswordSha256
	o.Encrypt = false
	result := new(updateStokReturn)
	json.Unmarshal(o.query(updateStokTemplate(o.User, o.hashedPassword), o.hostURL, false), &result)
	if result.ErrorCode == 0 && result.Result.StartSeq != nil {
		o.InsecureAuth = true
	} else {
		o.InsecureAuth = false
	}
}

func hash(value string) []byte {
	h := sha256.Sum256([]byte(value))
	return h[:]
}

func hashNHex(value string) string {
	return strings.ToUpper(fmt.Sprintf("%x", sha256.Sum256([]byte(value))))
}

func hashNHexOld(value string) string {
	return strings.ToUpper(fmt.Sprintf("%x", md5.Sum([]byte(value))))
}

func encodeAES(text []byte, key []byte, iv []byte) string {
	pad := aes.BlockSize - len(text)%aes.BlockSize
	bText := append(text, bytes.Repeat([]byte{byte(pad)}, pad)...)
	block, err := aes.NewCipher(key)
	if err != nil {
		p(err)
	}
	encoded := make([]byte, len(bText))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(encoded, bText)
	return string(encoded)
}

func decodeAES(text []byte, key []byte, iv []byte) string {
	decoded := text
	block, err := aes.NewCipher(key)
	if err != nil {
		p(err)
	}
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(decoded, decoded)
	return string(decoded)
}

func encodeB64(text string) string {
	return base64.StdEncoding.EncodeToString([]byte(text))
}

func decodeB64(text string) string {
	data, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return ""
	}
	return string(data)
}

func (o *Tapo) getDigestPasswd() (string, string) {
	result := new(loginInsecureResponse)
	err := json.Unmarshal(o.query(loginInitTemplate(o.User), o.hostURL, false), &result)
	if err != nil {
		p("Check! response struct outdated")
	}
	return hashNHex(o.hashedPassword + result.Result.Data.Nonce), result.Result.Data.Nonce
}

// Refresh stok. For authentication
func (o *Tapo) update() {
	if o.InsecureAuth {
		o.updateInsecure()
	} else {
		o.updateRaw()
	}
}

func (o *Tapo) updateInsecure() {
	o.hashedPassword = o.hashedPasswordSha256
	hashPass, nonce := o.getDigestPasswd()
	o.Key = hash("lsk" + nonce + hashPass)[:aes.BlockSize]
	o.Iv = hash("ivb" + nonce + hashPass)[:aes.BlockSize]
	o.Encrypt = true
	result := new(updateStokReturn)
	json.Unmarshal(o.query(loginNewTemplate(o.User, hashPass+nonce), o.hostURL, false), &result)
	if result.ErrorCode == 0 && result.Result.StartSeq != nil {
		//version >= 1.3.9(11)
		o.stokID = result.Result.Stok
		o.hostURLStok = o.hostURL + `/stok=` + o.stokID + `/ds`
		o.Seq = strconv.Itoa(*result.Result.StartSeq)
		o.userGroup = result.Result.UserGroup
	} else {
		p(`Authenticate failed. Try use another cred.`)
	}
}

func (o *Tapo) updateRaw() {
	o.hashedPassword = o.hashedPasswordMD5
	o.Encrypt = false
	result := new(updateStokReturn)
	json.Unmarshal(o.query(updateStokTemplate(o.User, o.hashedPassword), o.hostURL, false), &result)
	if result.ErrorCode == 0 && result.Result.StartSeq == nil {
		//version < 1.3.9(11)
		o.stokID = result.Result.Stok
		o.hostURLStok = o.hostURL + `/stok=` + o.stokID + `/ds`
		o.userGroup = result.Result.UserGroup
	} else {
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
}

// Get information about device tapo c200
func (o *Tapo) getDevice() {
	o.update()
	ret := o.query(manyTemplate(deviceInfoTemplate("")), o.hostURLStok, o.Encrypt)
	result := new(deviceRet)
	json.NewDecoder(bytes.NewReader(ret)).Decode(&result)
	o.deviceID = result.Result.Responses[0].Result.DeviceInfo.BasicInfo.DevID
	o.deviceModel = result.Result.Responses[0].Result.DeviceInfo.BasicInfo.DeviceModel
}

// Manual move
//
//	10 = 10 degree
//
// -10 = 10 degree reverse
func (o *Tapo) setMovePosition(x, y int) {
	o.update()
	o.query(movePositionTemplate(x, y), o.hostURLStok, o.Encrypt)
}

// Move action by X and Y
func (o *Tapo) setMoveAction() {
	x, _ := strconv.Atoi(o.Elements.MoveX)
	y, _ := strconv.Atoi(o.Elements.MoveY)
	o.setMovePosition(x, y)
}

// Get all making Presets in App
func (o *Tapo) getPresets() {
	o.update()
	ret := o.query(manyTemplate(presetConfigTemplate("")), o.hostURLStok, o.Encrypt)
	result := new(presetListReturn)
	json.NewDecoder(bytes.NewReader(ret)).Decode(&result)
	if len(result.Result.Responses) > 0 {
		o.Rotate = true
	}
	for _, v := range result.Result.Responses {
		for kk, vv := range v.Result.Preset.Preset.ID {
			o.presets = append(o.presets, &presets{ID: vv, Name: v.Result.Preset.Preset.Name[kk]})
		}
	}
}

// Switch to next preset
func (o *Tapo) setNextPreset() {
	if o.Rotate {
		o.update()
		o.lastPosition = o.rLast()
		if len(o.presets) > o.lastPosition+1 {
			o.lastPosition++
		} else {
			o.lastPosition = 0
		}
		next := o.presets[o.lastPosition]
		if o.Settings.PresetChangeOsd.Value {
			o.Settings.OsdText = next.Name
			o.Settings.VisibleOsdText.Value = true
			o.Settings.VisibleOsdTime.Value = true
			o.setOsdText()
		}
		o.query(
			nextPresetTemplate(next.ID),
			o.hostURLStok,
			o.Encrypt,
		)
		o.wLast(o.lastPosition)
	}
}

// Write log last file
func (o *Tapo) wLast(v int) {
	text := strconv.Itoa(v)
	os.WriteFile(filepath.Join(o.LastFile, o.deviceID+".last_preset"), []byte(text), 0775)
}

// Read log last file
func (o *Tapo) rLast() int {
	if lastDef, err := os.ReadFile(filepath.Join(o.LastFile, o.deviceID+".last_preset")); err == nil {
		last, _ := strconv.Atoi(string(lastDef))
		return last
	}
	os.Create(filepath.Join(o.LastFile, o.deviceID+".last_preset"))
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
	o.query(rebootTemplate(), o.hostURLStok, o.Encrypt)
}

// special function xBool
func (o *Types) xBool(s interface{}) *Types {
	switch val := s.(type) {
	case string:
		if _, err := strconv.Atoi(val); err == nil {
			ss := strings.Split(val, "")
			sslen := strconv.Itoa(len(ss))
			ssss := ss[len(ss)-1]
			tmp, _ := reflect.TypeOf(*new(Stages)).FieldByName("sBool" + sslen)
			if val, ex := tmp.Tag.Lookup(ssss); ex {
				o.Default = val
			}
		} else {
			o.Default = val
			o.Head = o.Default
			o.test()
			return o
		}
	case bool:
		switch val {
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
		ss := strings.Split(strconv.Itoa(val), "")
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

func sBool(value interface{}) bool {
	switch val := value.(type) {
	case string:
		switch val {
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
		return val
	default:
		fmt.Println("Error!")
		return false
	}
	return false
}

func (o *Tapo) getAlarm() (string, []string, string) {
	o.update()
	ret := o.query(manyTemplate(lastAlarmInfoTemplate("")), o.hostURLStok, o.Encrypt)
	result := new(lastAlarmInfoResponse)
	p(string(ret))
	json.NewDecoder(bytes.NewReader(ret)).Decode(&result)

	return result.Result.Responses[0].Result.MsgAlarm.Chn1MsgAlarmInfo.Enabled, result.Result.Responses[0].Result.MsgAlarm.Chn1MsgAlarmInfo.AlarmMode, result.Result.Responses[0].Result.MsgAlarm.Chn1MsgAlarmInfo.AlarmType
}

func (o *Tapo) updateAlarmSound() {
	o.update()
	enabled, list, alarmType := o.getAlarm()
	newList := []string{}
	for _, v := range list {
		if v != "sound" && enabled == "on" {
			newList = append(newList, v)
		}
	}

	if o.Elements.AlarmModeUpdateSound.Value {
		newList = append(newList, "sound")
		enabled = "on"
	}

	if len(newList) == 0 {
		enabled = "off"
		newList = []string{"sound", "light"}
	}

	o.query(
		alarmTemplate(
			alarmType,
			newList,
			enabled,
		),
		o.hostURLStok,
		o.Encrypt,
	)
}

func (o *Tapo) updateAlarmFlash() {
	o.update()
	enabled, list, alarmType := o.getAlarm()
	newList := []string{}
	for _, v := range list {
		if v != "light" && enabled == "on" {
			newList = append(newList, v)
		}
	}

	if o.Elements.AlarmModeUpdateFlash.Value {
		newList = append(newList, "light")
		enabled = "on"
	}

	if len(newList) == 0 {
		enabled = "off"
		newList = []string{"sound", "light"}
	}

	o.query(
		alarmTemplate(
			alarmType,
			newList,
			enabled,
		),
		o.hostURLStok,
		o.Encrypt,
	)
}

// Set alarm mode
// DetectEnableSound - include noise
// DetectSoundAlternativeMode - sound like a bip
// DetectEnableFlash - blinking led diode
func (o *Tapo) setAlarm() {
	o.update()

	list := []string{}

	if o.Settings.DetectEnableSound.Value && o.Settings.DetectEnableFlash.Value {
		list = []string{"sound", "light"}
	} else if !o.Settings.DetectEnableSound.Value && !o.Settings.DetectEnableFlash.Value {
		list = []string{"sound", "light"}
	} else if o.Settings.DetectEnableSound.Value {
		list = []string{"sound"}
	} else if o.Settings.DetectEnableFlash.Value {
		list = []string{"light"}
	}

	alarmType := "0"
	if o.Settings.DetectSoundAlternativeMode.Value {
		alarmType = "1"
	}

	o.query(
		alarmTemplate(
			alarmType,
			list,
			new(Types).xBool(o.Elements.AlarmMode.Value).Default,
		),
		o.hostURLStok,
		o.Encrypt,
	)
}

// Turn Indicator diode (red, green)
func (o *Tapo) setLedAction(value string) { //on off
	o.update()
	o.query(setLedTemplate(value), o.hostURLStok, o.Encrypt)
}

// Turn Indicator diode (red, green)
func (o *Tapo) setLed() {
	o.setLedAction(new(Types).xBool(o.Elements.Indicator.Value).Default)
}

// Get Time
func (o *Tapo) getTime() {
	o.update()
	result := new(getTimeRet)
	json.Unmarshal(o.query(getTimeTemplate(), o.hostURLStok, o.Encrypt), &result)
	o.TimeStr = result.System.ClockStatus.LocalTime
}

// Get Settings Image
func (o *Tapo) getImageSettings() {
	o.update()
	result := new(getImageSettingsRet)
	json.NewDecoder(bytes.NewReader(o.query(manyTemplate(ldcTemplate()), o.hostURLStok, o.Encrypt))).Decode(&result)
	o.FishEye = new(Types).xBool(result.Result.Responses[0].Result.Image.Switch.Ldc).isTrue
	o.Flip = result.Result.Responses[0].Result.Image.Switch.FlipType == "center"
}

// Set Correction
func (o *Tapo) setImageCorrection() {
	o.update()
	o.query(setImageCorrectionTemplate(new(Types).xBool(o.Elements.ImageCorrection.Value).Default), o.hostURLStok, o.Encrypt)
}

// Set Flip
func (o *Tapo) setImageFlip() {
	val := new(Types).xBool(o.Elements.ImageFlip.Value).Default
	if o.Elements.ImageFlip.Value {
		val = "center"
	}
	o.update()
	o.query(setImageFlipTemplate(val), o.hostURLStok, o.Encrypt)
}

// Motion detect with sensitivity
func (o *Tapo) getDetect() string {
	o.update()
	result := new(detectionConfigResponse)
	ret := o.query(manyTemplate(detectionConfigTemplate("")), o.hostURLStok, o.Encrypt)
	json.NewDecoder(bytes.NewReader(ret)).Decode(&result)
	return result.Result.Responses[0].Result.MotionDetection.MotionDet.Enabled
}

// Motion detect with sensitivity
func (o *Tapo) updateSens() {
	o.update()
	enabled := o.getDetect()
	p(string(o.query(detectTemplate(enabled, o.Settings.DetectSensitivity), o.hostURLStok, o.Encrypt)))
}

// Motion detect with sensitivity
func (o *Tapo) setDetect() {
	o.update()
	p(string(o.query(detectTemplate(new(Types).xBool(o.Elements.DetectMode.Value).Default, o.Settings.DetectSensitivity), o.hostURLStok, o.Encrypt)))
}

// Motion detect with sensitivity
func (o *Tapo) setDetectPerson() {
	o.update()
	o.query(setPersonDetectTemplate(new(Types).xBool(o.Elements.DetectPersonMode.Value).Default), o.hostURLStok, o.Encrypt)
}

// Turn camera in private mode with stop video channel
func (o *Tapo) setPrivacy() {
	o.update()
	o.query(privacyTemplate(new(Types).xBool(o.Elements.PrivacyMode.Value).Default), o.hostURLStok, o.Encrypt)
}

// Turn irc flashlight
func (o *Tapo) setNightMode() {
	o.update()
	o.query(nightModeTemplate(new(Types).xBool(o.Elements.NightMode.Value).Default), o.hostURLStok, o.Encrypt)
}

// Turn irc flashlight
func (o *Tapo) setNightModeAuto() {
	if o.Elements.NightModeAuto.Value {
		o.update()
		o.query(nightModeTemplate("auto"), o.hostURLStok, o.Encrypt)
	}
}

// Autotracking all motion. BETA
func (o *Tapo) setAutotracking() {
	o.update()
	o.query(autotrackingTemplate(new(Types).xBool(o.Elements.AutotrackingMode.Value).Default), o.hostURLStok, o.Encrypt)
}

// get Text OSD
func (o *Tapo) getOsd() (string, string) {
	o.update()
	result := new(getOSDRet)
	json.NewDecoder(bytes.NewReader(o.query(getOSDTemplate(), o.hostURLStok, o.Encrypt))).Decode(&result)
	if len(o.Settings.OsdText) == 0 {
		o.Settings.OsdText = result.OSD.LabelInfo[0].LabelInfo1.Text
	}
	return result.OSD.LabelInfo[0].LabelInfo1.Enabled, result.OSD.Date.Enabled
}

// Text OSD
func (o *Tapo) setOsdTime() {
	textEnabled, _ := o.getOsd()
	o.update()
	o.query(
		osdTemplate(
			new(Types).xBool(o.Settings.VisibleOsdTime.Value).Default,
			textEnabled,
			o.Settings.OsdText,
		),
		o.hostURLStok,
		o.Encrypt,
	)
}

// Text OSD
func (o *Tapo) setOsdText() {
	_, timeEnabled := o.getOsd()

	if len(o.Settings.OsdText) > 16 {
		//16 symbols, not bytes
		o.Settings.OsdText = string([]rune(o.Settings.OsdText)[0:16])
	}

	o.update()
	o.query(
		osdTemplate(
			timeEnabled,
			new(Types).xBool(o.Settings.VisibleOsdText.Value).Default,
			o.Settings.OsdText,
		),
		o.hostURLStok,
		o.Encrypt,
	)
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
