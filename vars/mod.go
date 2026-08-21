package vars

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	VectorResources = "/anki/data/assets/cozmo_resources/"
	WiredData       = "/data/wired/mods/"
)

type Modification interface {
	Name() string
	Description() string
	HTTP(w http.ResponseWriter, r *http.Request)
	Load() error
}

var EnabledMods []Modification

func IsEndpoint(r *http.Request, endpoint string) bool {
	return strings.Contains(r.URL.Path, endpoint)
}

func FindMod(name string) (Modification, error) {
	for index, mod := range EnabledMods {
		if strings.TrimSpace(name) == mod.Name() {
			return EnabledMods[index], nil
		}
	}
	return nil, errors.New("mod not found")
}

func GetModDir(modname string) string {
	return filepath.Join(WiredData, modname)
}

const ankiVictorData = "/data/data/com.anki.victor"

func SaveFile(contents string, path string) error {
	os.MkdirAll(filepath.Dir(path), 0777)
	err := os.WriteFile(path, []byte(contents), 0777)
	if strings.HasPrefix(path, ankiVictorData) {
		SetAnkiPerms()
	}
	return err
}

func ReadFile(path string) (contents string, err error) {
	out, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// SetAnkiPerms makes Anki's /data tree writable by engine/net after wired
// (root) MkdirAll. Otherwise CLEAR OUT SOUL leaves persistent/ as root:root
// 755, vic-switchboard cannot create sessions, and the robot loops fault 913.
func SetAnkiPerms() {
	persistent := filepath.Join(ankiVictorData, "persistent")
	switchboard := filepath.Join(persistent, "switchboard")
	cache := filepath.Join(ankiVictorData, "cache")
	for _, d := range []string{ankiVictorData, persistent, switchboard, cache} {
		_ = os.MkdirAll(d, 0770)
	}
	_ = exec.Command("chown", "anki:anki", ankiVictorData, persistent, cache).Run()
	_ = exec.Command("chmod", "770", ankiVictorData, persistent, cache, switchboard).Run()
	_ = exec.Command("chown", "-R", "net:anki", switchboard).Run()
	if _, err := os.Stat(filepath.Join(persistent, "wake_engine")); err == nil {
		_ = exec.Command("chown", "anki:anki", filepath.Join(persistent, "wake_engine")).Run()
	}
}

func ExtraHTTP(w http.ResponseWriter, r *http.Request) {
	if IsEndpoint(r, "restartvic") {
		RestartVic()
		HTTPSuccess(w, r)
	} else {
		HTTPError(w, r, "not found")
	}
}

func InitMods() {
	for _, mod := range EnabledMods {
		fmt.Println("Loading " + mod.Name() + "...")
		err := mod.Load()
		if err != nil {
			fmt.Println("ERROR loading", mod.Name(), ":", err)
			continue
		}
		http.HandleFunc("/api/mods/"+mod.Name()+"/", mod.HTTP)
	}
	SetAnkiPerms()
	http.HandleFunc("/api/extra/", ExtraHTTP)
}

func StopVic() {
	// Behavior("DevBaseBehavior")
	time.Sleep(time.Second * 1)
	exec.Command("/bin/bash", "-c", "systemctl stop anki-robot.target").Output()
	time.Sleep(time.Second * 4)
}

func StartVic() {
	exec.Command("/bin/bash", "-c", "systemctl start anki-robot.target").Output()
	time.Sleep(time.Second * 3)
}

func RestartVic() {
	StopVic()
	StartVic()
}

type HTTPStatus struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func HTTPSuccess(w http.ResponseWriter, r *http.Request) {
	var status HTTPStatus
	status.Status = "success"
	successBytes, _ := json.Marshal(status)
	w.Write(successBytes)
}

func HTTPError(w http.ResponseWriter, r *http.Request, err string) {
	var status HTTPStatus
	status.Status = "error"
	status.Message = err
	errorBytes, _ := json.Marshal(status)
	w.WriteHeader(500)
	w.Write(errorBytes)
}

// type BehaviorMessage struct {
// 	Type   string `json:"type"`
// 	Module string `json:"module"`
// 	Data   struct {
// 		BehaviorName     string `json:"behaviorName"`
// 		PresetConditions bool   `json:"presetConditions"`
// 	} `json:"data"`
// }

// //{"type":"data","module":"behaviors","data":{"behaviorName":"DevBaseBehavior","presetConditions":false}}

// func Behavior(behavior string) {
// 	u := url.URL{Scheme: "ws", Host: "localhost:8888", Path: "/socket"}

// 	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
// 	if err != nil {
// 		fmt.Println("dial:", err)
// 		return
// 	}
// 	defer c.Close()

// 	message := BehaviorMessage{
// 		Type:   "data",
// 		Module: "behaviors",
// 		Data: struct {
// 			BehaviorName     string `json:"behaviorName"`
// 			PresetConditions bool   `json:"presetConditions"`
// 		}{
// 			BehaviorName:     behavior,
// 			PresetConditions: false,
// 		},
// 	}

// 	marshaledMessage, err := json.Marshal(message)
// 	if err != nil {
// 		log.Fatal("marshal:", err)
// 	}

// 	err = c.WriteMessage(websocket.TextMessage, marshaledMessage)
// 	if err != nil {
// 		log.Fatal("write:", err)
// 	}
// }
