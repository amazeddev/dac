package execute

import (
	"dac/client"
	"dac/parser"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type DataArgs struct {
	Name string						`json:"name,omitempty"`
	Key string						`json:"key,omitempty"`
	Tabname string				`json:"tabname,omitempty"`
	Results []string			`json:"results,omitempty"`
	Columns []string			`json:"columns,omitempty"`
	Path string						`json:"path,omitempty"`
}

func RunChain(homeDir string, args parser.CalcChain) ([]byte, error) {
	data, err := json.Marshal(args)
	if err != nil {
		return []byte{}, err
	}

	cmd := exec.Command("bash", "-c", fmt.Sprintf("source %[1]v/.dac/.env/bin/activate && python3 %[1]v/.dac/lib/chain.py '%v'", homeDir, string(data)))
	
	out, err := cmd.Output()
	if err != nil {
		return []byte{}, err
	}
	return out, nil
}

func RunData(homeDir string, method string, args DataArgs) ([]byte, error) {
	data, err := json.Marshal(args)
	if err != nil {
		return []byte{}, err
	}

	cmd := exec.Command("bash", "-c", fmt.Sprintf("source %[1]v/.dac/.env/bin/activate && python3 %[1]v/.dac/lib/data.py %v '%v'", homeDir, method, string(data)))
	out, err := cmd.Output()
	if err != nil {
		fmt.Println("Error: ", err)
	}
	if err != nil {
		return []byte{}, err
	}
	return out, nil
}

func RunConfig(homeDir string) ([]byte, error) {

	cmd := exec.Command("bash", "-c", fmt.Sprintf("python3 -m virtualenv %v/.dac/.env", homeDir))
	_, err := cmd.Output()
	if err != nil {
		return []byte{}, err
	}

	cmd = exec.Command("bash", "-c", fmt.Sprintf("source %[1]v/.dac/.env/bin/activate && python3 -m pip install -r %[1]v/.dac/requirements.txt", homeDir))
	out, err := cmd.Output()
	if err != nil {
		return []byte{}, err
	}
	if err != nil {
		return []byte{}, err
	}
	return out, nil
}

func DownloadExtract(fileUrl, homeDir string) ([]byte, error) {
	var (
		fileName    string
	)
	// Build fileName from fullPath
	fileURL, err := url.Parse(fileUrl)
	if err != nil {
		return []byte{}, err
	}
	path := fileURL.Path
	segments := strings.Split(path, "/")
	fileName = segments[len(segments)-1]

	// Create blank file
	file, err := os.Create(fmt.Sprintf("%v/.dac/%v", homeDir, fileName))
	if err != nil {
		return []byte{}, err
	}
	client := http.Client{
			CheckRedirect: func(r *http.Request, via []*http.Request) error {
					r.URL.Opaque = r.URL.Path
					return nil
			},
	}
	// Put content on file
	resp, err := client.Get(fileUrl)
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()

	size, _ := io.Copy(file, resp.Body)

	defer file.Close()

	fmt.Printf("Downloaded a file %s with size %d\n\n", fileName, size)
	time.Sleep(1 * time.Second)

	cmd := exec.Command("bash", "-c", fmt.Sprintf(" cd %[1]v/.dac && tar -xf %[2]v && rm %[2]v", homeDir, fileName))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	err = cmd.Run()
	if err != nil {
		return []byte{}, err
	}
	return []byte{}, nil
}

func StopEngine(kvs client.Rpc_client) error {
	reply, err := kvs.Info()
	if err != nil {
		return err
	}
	cmd := exec.Command("kill", "-9", reply.Pid)
	err = cmd.Run()
	if err != nil {
		return err
	}
	fmt.Println("project engine closed...")

	return nil
}

func StartEngin(kvs *client.Rpc_client, name string, home string) error {
		err := kvs.Connect()
		if err != nil {
			fmt.Print("(re)initializing project engine...")
			cmd := exec.Command(fmt.Sprintf("%v/%v", home, ".dac/kvstorage"))
			cmd.Env = os.Environ()
			cmd.Stdin = nil
			cmd.Stdout = nil
			cmd.Stderr = nil
			cmd.ExtraFiles = nil
			cmd.SysProcAttr = &syscall.SysProcAttr{
				Setsid: true,
			}

			if err := cmd.Start(); err != nil {
				return err
			}

			time.Sleep(2 * time.Second)

			err = kvs.Connect()
			if err != nil {
				return err
			}

			_, err := kvs.SetPid(client.PidArgs{Name: name, Pid: fmt.Sprint(cmd.Process.Pid)})
			if err != nil {
				return err
			}
			fmt.Printf("DONE!\n")
		}
		return nil
}


