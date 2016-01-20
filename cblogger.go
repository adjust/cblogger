package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	rw             *RotateWriter
	logDir         string
	logFile        string
	rotateInterval int64
	port           int
	timeFormat     string
	s3space        string
	timeLocation   string
	logFilePrefix  string = "adjust_logs_"

	// logged parameters, the order in the slice is the order
	// in which they will appear in the csv
	paramList = []string{
		"app_name", "event_name",
	}
)

// init globals from cli options
func init() {
	flag.StringVar(&logDir, "logdir", "./", "log Directory")
	flag.StringVar(&logFile, "logfile", "csv", "log file")
	flag.StringVar(&timeFormat, "time", "2006-01-02_15", "time format string")
	flag.StringVar(&timeLocation, "location", "UTC", "time location string")
	flag.StringVar(&s3space, "s3", "", "s3space name")
	flag.Int64Var(&rotateInterval, "interval", 3600, "rotation interval")
	flag.IntVar(&port, "port", 3000, "server port")
	flag.Parse()
}

// Implementation of writer interface with support for file rotations
type RotateWriter struct {
	lock     sync.Mutex
	dirname  string
	filename string
	fp       *os.File
}

func New(dirname string, filename string) *RotateWriter {
	var err error
	w := &RotateWriter{dirname: dirname, filename: filename}
	w.fp, err = os.OpenFile(dirname+"/"+filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	return w
}

func (w *RotateWriter) Write(output []byte) (int, error) {
	w.lock.Lock()
	defer w.lock.Unlock()
	return w.fp.Write(output)
}

func (w *RotateWriter) Rotate() (file string, err error) {
	w.lock.Lock()
	defer w.lock.Unlock()

	if w.fp != nil {
		err = w.fp.Close()
		w.fp = nil
		if err != nil {
			return
		}
	}

	_, err = os.Stat(w.dirname + "/" + w.filename)
	if err == nil {
		pst, time_err := time.LoadLocation("UTC")
		if time_err != nil {
			log.Println(time_err)
		}
		file = w.dirname + "/" + logFilePrefix + time.Now().In(pst).Add(-time.Hour).Format(timeFormat) + ".csv"
		// Some timezones have leap hours. In this case we need to be able to have two csv files for the same hour
		if _, err := os.Stat(file); err == nil {
			file = w.dirname + "/" + logFilePrefix + time.Now().In(pst).Add(-time.Hour).Format(timeFormat) + "_2.csv"
		}
		err = os.Rename(w.dirname+"/"+w.filename, file)
		if err != nil {
			return
		}
	}

	w.fp, err = os.Create(w.dirname + "/" + w.filename)
	return
}

// handle function for all incoming requests
func rootHandler(w http.ResponseWriter, r *http.Request) {
	result := ""

	paramMap := map[string]string{}

	// first parse the paramList and add that to result
	for _, param := range paramList {
		value := r.FormValue(param)
		if value != "" {
			paramMap[param] = value
		}
	}

	// now add label params from the label param to the result
	xLabelParse(r.FormValue("label"), paramMap)

	for _, param := range paramList {
		if paramMap[param] != "" {
			result += fmt.Sprintf("\"%s\",", strings.Replace(paramMap[param], "\"", "\\\"", -1))
		} else if paramMap[strings.ToLower(param)] != "" {
			result += fmt.Sprintf("\"%s\",", strings.Replace(paramMap[strings.ToLower(param)], "\"", "\\\"", -1))
		} else {
			result += ","
		}
	}

	// write the result to the csv file
	rw.Write([]byte(fmt.Sprintf("%s\n", strings.TrimSuffix(result, ","))))

	// answer to the client
	fmt.Fprintf(w, "OK")
}

// parse the label param and extract the label params
func xLabelParse(args string, paramMap map[string]string) {
	pairs := []string{}
	if args != "" {
		pairs = strings.Split(args, "&")
	}

	for _, pair := range pairs {
		keyValue := strings.Split(pair, "=")
		if len(keyValue) > 1 {
			param := strings.ToLower(keyValue[0])
			paramMap[param] = keyValue[1]
		}
	}
}

// background process that rotates the files every rotateInterval (1 hour default)
// -> old file is renamed
// -> new file is created
// -> old file is uploaded to s3
// -> only last 12 csv file are kept locally
func watchdog() {
	timestamp := time.Now().Unix()
	time.Sleep(time.Duration(rotateInterval-(timestamp%rotateInterval)) * time.Second)
	ticker := time.NewTicker(time.Duration(rotateInterval) * time.Second)
	file, err := rw.Rotate()
	if err != nil {
		panic(err)
	}
	archiveCmd(file)
	cleanOldFiles()

	for ts := range ticker.C {
		file, err := rw.Rotate()
		if err != nil {
			panic(err)
		}
		archiveCmd(file)
		log.Println(ts)
	}
}

// put the file to s3, log errors
func archiveCmd(file string) {
	if s3space == "" {
		return
	}
	var myArgs []string = []string{}
	myArgs = append(myArgs, "s3")
	myArgs = append(myArgs, "cp")
	myArgs = append(myArgs, file)
	myArgs = append(myArgs, fmt.Sprintf("s3://%s/", s3space))

	fmt.Println("aws", myArgs)
	out, err := exec.Command("aws", myArgs...).CombinedOutput()

	if err != nil {
		fmt.Println(string(out), err.Error())
		basename := path.Base(file)
		// if the upload failed rename the csv to avoid purging on cleanup
		err = os.Rename(file, logDir+"/failed_"+basename)
		if err != nil {
			fmt.Println(err)
		}
	}
	cleanOldFiles()
}

// only keep the last 12 csv's locally
func cleanOldFiles() {
	var files sort.StringSlice
	var err error

	files, err = filepath.Glob(logDir + "/" + logFilePrefix + "*")
	if err != nil {
		fmt.Println(err)
		return
	}

	if len(files) > 12 {
		for _, file := range files[:len(files)-12] {
			fmt.Printf("Deleting %s\n", file)
			err := os.Remove(file)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func main() {
	go watchdog()
	rw = New(logDir, logFile)
	http.HandleFunc("/", rootHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
