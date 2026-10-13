package mods

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// English-teaching course preset (identity_mode "course"), ported from
// ubt_speech_demo Self-Control. Mirrors vic-cloud internal/xiaozhi/course.
const xiaozhiIdentityCourse = "course"

type xzCourseVoice struct {
	Label string   `json:"label"`
	Pool  []string `json:"-"`
}

type xzCourseBook struct {
	Name  string   `json:"name"`
	Units []string `json:"units"`
}

type xzCourse struct {
	Idx    int              `json:"idx"`
	ID     string           `json:"id"`
	Name   string           `json:"name"`
	Voices [2]xzCourseVoice `json:"voices"`
	Books  []xzCourseBook   `json:"books"`
}

func xzCoursePool(prefix string) []string {
	out := make([]string, 0, 10)
	for i := 0; i < 10; i++ {
		out = append(out, prefix+":1"+strconv.Itoa(i))
	}
	return out
}

func xzNumberedUnits(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = "Unit " + strconv.Itoa(i+1)
	}
	return out
}

var xzCourses = []xzCourse{
	{
		Idx: 1, ID: "explorers", Name: "Explorers",
		Voices: [2]xzCourseVoice{
			{"Song ngữ Anh – Việt", xzCoursePool("ba:53:9e:c5:ba")},
			{"Tiếng Anh", xzCoursePool("ba:53:9e:c5:fe")},
		},
		Books: []xzCourseBook{
			{"Pre-Discovery A", []string{
				"Unit Routines", "Unit Welcome The Magic Forest", "Unit One Let's Draw",
				"Unit Two Let's Play", "Unit Review One", "Unit Three The Big Monster",
				"Unit Special Day One", "Unit Special Day Two",
			}},
			{"Pre-Discovery B", []string{
				"Unit Routines Review", "Unit Four My Family", "Unit Review Two",
				"Unit Five Where's My Bird?", "Unit Midterm Test", "Unit Six Let's Tidy Up!",
				"Unit Review Three", "Unit Special Day Three", "Unit Special Day Four",
				"Unit Review Four", "Unit Final Test",
			}},
			{"PRE-SPARK A", []string{
				"Unit Routines", "Unit Welcome Let's Remember", "Unit One The Surprise",
				"Unit Two The Brown Mouse", "Unit Review One", "Unit Three Where Is Greenman?",
				"Unit Special Day One", "Unit Special Day Two",
			}},
			{"SPARK Two A", []string{
				"Unit Our Friends", "Unit One Weather", "Unit Two Our Families",
				"Unit Three Our Places", "Unit Four Our Colors and Shapes",
			}},
			{"SPARK Three A", []string{
				"Unit Hello Again", "Unit One Our Day", "Unit Two Dinner Time",
				"Unit Three With My Friends", "Unit Four Our Colors and Shapes",
				"Unit Four Our Faces",
			}},
		},
	},
	{
		Idx: 2, ID: "younginnovators", Name: "Young Innovators",
		Voices: [2]xzCourseVoice{
			{"Tiếng Việt", xzCoursePool("ba:53:9e:c5:ef")},
			{"Tiếng Anh", xzCoursePool("ba:53:9e:c6:fe")},
		},
		Books: []xzCourseBook{
			{"QUEST One A", []string{
				"Unit One Hello", "Unit Two My School", "Unit Three Favourite Toys",
				"Unit Four My Family", "Unit Five Our Pets", "Unit Six My Face",
			}},
			{"QUEST Two B", []string{
				"Unit Seven At the Farm", "Unit Eight My Town", "Unit Nine Our Clothes",
				"Unit Ten Our Hobbies", "Unit Eleven My Birthday", "Unit Twelve Our Holiday",
			}},
			{"QUEST Three A", []string{
				"Unit Hello", "Unit One Family Matters", "Unit Two Home Sweet Home",
				"Unit Values One and Two", "Unit Three A Day in the Life", "Unit Four In the City",
				"Unit Values Three and Four", "Unit YLE Movers Listening Skills Practice",
			}},
		},
	},
	{
		Idx: 3, ID: "futureleaders", Name: "Future Leaders",
		Voices: [2]xzCourseVoice{
			{"Tiếng Việt", xzCoursePool("ba:53:9e:c7:ef")},
			{"Tiếng Anh", xzCoursePool("ba:53:9e:c8:ef")},
		},
		Books: []xzCourseBook{
			{"Summit One A", []string{
				"Unit Welcome", "Unit One Having a Good Time", "Unit Two Spending Money",
				"Unit Three We Are What We Eat", "Unit Four All in the Family",
				"Unit Five No Place Like Home", "Unit Six Friends Forever",
			}},
		},
	},
	{
		Idx: 4, ID: "ielts", Name: "IELTS",
		Voices: [2]xzCourseVoice{
			{"Tiếng Việt", xzCoursePool("ba:53:9e:c2:ef")},
			{"Tiếng Anh", xzCoursePool("ba:53:9e:c3:ef")},
		},
		Books: []xzCourseBook{{"IELTS", xzNumberedUnits(6)}},
	},
	{
		Idx: 5, ID: "toeic", Name: "TOEIC",
		Voices: [2]xzCourseVoice{
			{"Tiếng Việt", xzCoursePool("ba:53:9e:c9:ef")},
			{"Tiếng Anh", xzCoursePool("ba:53:9e:c1:ef")},
		},
		Books: []xzCourseBook{{"TOEIC", xzNumberedUnits(6)}},
	},
}

const xzDefaultCourseIdx = 1

func xzCourseByIdx(idx int) (xzCourse, bool) {
	for _, c := range xzCourses {
		if c.Idx == idx {
			return c, true
		}
	}
	return xzCourse{}, false
}

func isXiaozhiCourse(cfg XiaozhiCfg) bool {
	return strings.EqualFold(strings.TrimSpace(cfg.IdentityMode), xiaozhiIdentityCourse)
}

func xzCourseKey(idx int) string { return strconv.Itoa(idx) }

func xzCourseUnitKey(idx, book int) string { return strconv.Itoa(idx) + "." + strconv.Itoa(book) }

// xzCourseLesson is the stored lesson for cfg's course (clamped; book 0-based, unit 1-based).
func xzCourseLesson(cfg XiaozhiCfg) (idx, voice, book, unit int) {
	c, ok := xzCourseByIdx(cfg.CourseIdx)
	if !ok {
		c, _ = xzCourseByIdx(xzDefaultCourseIdx)
	}
	idx = c.Idx
	if cfg.CourseVoice[xzCourseKey(idx)] == 1 {
		voice = 1
	}
	book = cfg.CourseBook[xzCourseKey(idx)]
	if book < 0 || book >= len(c.Books) {
		book = 0
	}
	unit = cfg.CourseUnit[xzCourseUnitKey(idx, book)]
	if n := len(c.Books[book].Units); unit < 1 || unit > n {
		unit = 1
	}
	return idx, voice, book, unit
}

func xzPickCourseMAC(idx, voice int, avoid string) string {
	c, ok := xzCourseByIdx(idx)
	if !ok {
		return ""
	}
	pool := c.Voices[voice&1].Pool
	avoid = strings.ToLower(strings.TrimSpace(avoid))
	var b [1]byte
	_, _ = rand.Read(b[:])
	start := int(b[0]) % len(pool)
	for i := 0; i < len(pool); i++ {
		if m := pool[(start+i)%len(pool)]; m != avoid {
			return m
		}
	}
	return pool[start]
}

func xzMACInCoursePool(idx, voice int, mac string) bool {
	c, ok := xzCourseByIdx(idx)
	if !ok {
		return false
	}
	mac = strings.ToLower(strings.TrimSpace(mac))
	for _, m := range c.Voices[voice&1].Pool {
		if m == mac {
			return true
		}
	}
	return false
}

// applyXiaozhiCourseIdentity switches cfg to the course pool. A new MAC/UUID
// is drawn when forced (course or voice changed) or the MAC is not in the pool.
func applyXiaozhiCourseIdentity(cfg *XiaozhiCfg, force bool) {
	cfg.IdentityMode = xiaozhiIdentityCourse
	idx, voice, _, _ := xzCourseLesson(*cfg)
	cfg.CourseIdx = idx
	cfg.OTABaseURL = xiaozhiTenclassOTA
	cfg.Endpoint = xiaozhiTenclassWSS
	cfg.AutoApplyOTAWebsocket = true
	cfg.TTSMode = "xiaozhi"
	if cfg.ConversationMode == "" {
		cfg.ConversationMode = "continuous"
	}
	if force || !xzMACInCoursePool(idx, voice, cfg.DeviceID) {
		cfg.DeviceID = xzPickCourseMAC(idx, voice, cfg.DeviceID)
		cfg.ClientID = genUUIDv4()
		cfg.Token = ""
	}
}

// xzCourseUpdate is a set_course request; nil fields keep their stored value.
type xzCourseUpdate struct {
	Idx, Voice, Book, Unit *int
	StudentName            *string
}

// applyXiaozhiCourseUpdate stores the lesson and reports whether the device
// identity changed (course or voice), which needs a vic-cloud apply.
func applyXiaozhiCourseUpdate(cfg *XiaozhiCfg, u xzCourseUpdate) (identityChanged bool) {
	wasCourse := isXiaozhiCourse(*cfg)
	oldIdx, oldVoice, _, _ := xzCourseLesson(*cfg)
	if u.StudentName != nil {
		cfg.StudentName = strings.TrimSpace(*u.StudentName)
	}
	if u.Idx != nil {
		if _, ok := xzCourseByIdx(*u.Idx); ok {
			cfg.CourseIdx = *u.Idx
		}
	}
	idx, _, _, _ := xzCourseLesson(*cfg)
	cfg.CourseIdx = idx
	if cfg.CourseVoice == nil {
		cfg.CourseVoice = map[string]int{}
	}
	if cfg.CourseBook == nil {
		cfg.CourseBook = map[string]int{}
	}
	if cfg.CourseUnit == nil {
		cfg.CourseUnit = map[string]int{}
	}
	if u.Voice != nil {
		cfg.CourseVoice[xzCourseKey(idx)] = *u.Voice & 1
	}
	if u.Book != nil {
		cfg.CourseBook[xzCourseKey(idx)] = *u.Book
	}
	_, _, book, _ := xzCourseLesson(*cfg)
	cfg.CourseBook[xzCourseKey(idx)] = book
	if u.Unit != nil {
		cfg.CourseUnit[xzCourseUnitKey(idx, book)] = *u.Unit
	}
	_, voice, _, unit := xzCourseLesson(*cfg)
	cfg.CourseUnit[xzCourseUnitKey(idx, book)] = unit

	identityChanged = !wasCourse || idx != oldIdx || voice != oldVoice || !cfg.Enabled
	cfg.Enabled = true
	applyXiaozhiCourseIdentity(cfg, identityChanged)
	return identityChanged
}

// xiaozhiConnectionChanged: fields that need a new WSS session (or the
// Xiaozhi/Vosk switch) to take effect.
func xiaozhiConnectionChanged(a, b XiaozhiCfg) bool {
	return a.Enabled != b.Enabled || a.DeviceID != b.DeviceID || a.ClientID != b.ClientID ||
		a.Endpoint != b.Endpoint || a.Token != b.Token || a.OTABaseURL != b.OTABaseURL ||
		a.ProtocolVersion != b.ProtocolVersion || !strings.EqualFold(a.IdentityMode, b.IdentityMode)
}

// --- apply config in vic-cloud ---

// Set by vic-cloud builds that apply xiaozhi.json in place (see its reload.go).
var (
	xiaozhiReloadFlagPath      = "/run/vic-cloud/xiaozhi-reload"
	xiaozhiReloadSupportedPath = "/run/vic-cloud/xiaozhi-reload-supported"
)

var (
	restartMu    sync.Mutex
	restartTimer *time.Timer
	// Overridable in tests.
	restartVicCloudFn = restartVicCloud
)

// Coalesces quick toggles into one restart: a second restart while vic-cloud
// is still loading Vosk used to end in fault 923.
const vicCloudRestartDebounce = 2 * time.Second

func scheduleVicCloudRestart() {
	restartMu.Lock()
	defer restartMu.Unlock()
	if restartTimer != nil {
		restartTimer.Stop()
	}
	restartTimer = time.AfterFunc(vicCloudRestartDebounce, func() {
		restartMu.Lock()
		restartTimer = nil
		restartMu.Unlock()
		restartVicCloudFn()
	})
}

// applyVicCloudConfig asks vic-cloud to pick up xiaozhi.json: in place (close
// the WSS, reconnect, greet) when supported, otherwise a debounced restart.
// Returns true when applied in place.
func applyVicCloudConfig() bool {
	if _, err := os.Stat(xiaozhiReloadSupportedPath); err == nil {
		if err := os.MkdirAll(filepath.Dir(xiaozhiReloadFlagPath), 0775); err == nil {
			if err := os.WriteFile(xiaozhiReloadFlagPath, []byte("1"), 0664); err == nil {
				return true
			}
		}
	}
	scheduleVicCloudRestart()
	return false
}
