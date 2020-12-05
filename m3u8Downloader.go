package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func DownloadFile(filepath string, url string) error {

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("1 error(DownloadFile)", err)
		return err
	}
	defer resp.Body.Close()

	// Create the ts file
	out, err := os.Create(filepath)
	if err != nil {
		fmt.Println("2 error(DownloadFile)", err)
		return err
	}
	defer out.Close()

	// Write the ts bytes to file
	_, err = io.Copy(out, resp.Body)
	return err

}
func getM3u8TsList(m3u8URL string) []string {
	response, err := http.Get(m3u8URL)
	if err != nil {
		log.Fatal(err)
		fmt.Println("error(getM3u8TsList)", err)
		return nil
	}
	if response.StatusCode != 200 {
		log.Fatal("getEdgeTsIndex response code not 200")
		fmt.Println("getEdgeTsIndex response code not 200")
		fmt.Println("error(getM3u8TsList)", err)
		return nil
	}
	// Create a goquery document from the HTTP response
	document, err := goquery.NewDocumentFromReader(response.Body)
	response.Body.Close()

	if err != nil {
		log.Fatal("getEdgeTsIndex Error loading HTTP response body. ", err)
		fmt.Println(err)
		fmt.Println("error(getM3u8TsList)", err)
		return nil
	}

	lines := strings.Split(document.Text(), "\n")
	tsList := make([]string, 0)
	for i := 0; i < len(lines); i++ {
		if strings.Contains(lines[i], ".ts") {
			tsList = append(tsList, lines[i])
		}
	}

	return tsList
}

var clear map[string]func() //create a map for storing clear funcs

func init() {
	clear = make(map[string]func()) //Initialize it
	clear["linux"] = func() {
		cmd := exec.Command("clear") //Linux example, its tested
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
	clear["windows"] = func() {
		cmd := exec.Command("cmd", "/c", "cls") //Windows example, its tested
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

func CallClear() {
	value, ok := clear[runtime.GOOS] //runtime.GOOS -> linux, windows, darwin etc.
	if ok {                          //if we defined a clear func for that platform:
		value() //we execute it
	} else { //unsupported platform
		panic("Your platform is unsupported! I can't clear terminal screen :(")
	}
}

// exists returns whether the given file or directory exists
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func mergeTsFiles(tsAmount int, tempFolderName string) {
	//returns the current working dir
	wd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	//list of temp output ts files to be merge to a final temp ts file.
	ffmpegOutputTsFiles := "concat:"
	//the number of ts files we merege per merge command
	tsPerConcat := 10
	for i := 0; i < tsAmount; i += tsPerConcat {
		ffmpegTsFiles := "concat:"
		for j := i; (j < i+tsPerConcat) && (j < tsAmount); j++ {
			//ts file
			ffmpegTsFiles = fmt.Sprintf("%s%d.ts|", ffmpegTsFiles, j)
		}
		//removes the last | character.
		ffmpegTsFiles = ffmpegTsFiles[0 : len(ffmpegTsFiles)-1]
		outputTs := fmt.Sprintf("output_%d.ts", i)

		ffmpegOutputTsFiles = fmt.Sprintf("%soutput_%d.ts|", ffmpegOutputTsFiles, i)

		cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "panic", "-i", ffmpegTsFiles, "-c", "copy", outputTs)
		cmd.Dir = fmt.Sprintf("%s\\%s", wd, tempFolderName)
		err = cmd.Run()
		cmd.Wait()
		if err != nil {
			log.Printf("Command finished with error: %v", err)
			os.Exit(1)
		}

	}
	//removes the last | character.
	ffmpegOutputTsFiles = ffmpegOutputTsFiles[0 : len(ffmpegOutputTsFiles)-1]

	//merges the temp output ts files to one output ts file(which will be later convert to mp4 file).
	//the rationale in the approach of multiple meregers, is that there is an input length limit defined by ffmpeg.
	//therefore I tried to reduce the input length.
	//Note:it is not a perfect solution because for a much large amount of ts files,
	//     we will be ended with a long input length, its can be solved.
	//     but for now, I prefer to leave it as it is, because it is covers the rational amount of ts files(in my opinon).
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "panic", "-i", ffmpegOutputTsFiles, "-c", "copy", "output.ts")
	cmd.Dir = fmt.Sprintf("%s\\%s", wd, tempFolderName)
	err = cmd.Run()
	cmd.Wait()
	if err != nil {
		log.Printf("Command finished with error: %v", err)
		os.Exit(1)
	}

}
func saveFileAsMp4Format(tempFolderName, dstFolder, fileName string) {
	//returns the current working dir
	wd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	//save final ts file as to mp4 format
	cmd := exec.Command("ffmpeg", "-i", "output.ts", "-map", "0", "-c", "copy", fmt.Sprintf("%s\\%s.mp4", dstFolder, fileName))
	cmd.Dir = fmt.Sprintf("%s\\%s", wd, tempFolderName)
	err = cmd.Run()
	cmd.Wait()
	if err != nil {
		log.Printf("Command finished with error: %v", err)
		os.Exit(1)
	}
}

func main() {

	if len(os.Args) < 4 {
		fmt.Println("pararmets error: you have to pass file name(to be save), m3u8 url, folder destination respectively")
		return
	}
	fmt.Println("name:", os.Args[0], "filename:", os.Args[1])
	fmt.Println("href:", os.Args[2])
	fmt.Println("destination folder:", os.Args[3])
	fileName := os.Args[1]
	m3u8URL := os.Args[2]
	dstFolder := os.Args[3]
	//removes the last character from dstFolder if its backslash
	for string(dstFolder[len(dstFolder)-1]) == "\\" {
		dstFolder = dstFolder[0 : len(dstFolder)-1]
	}
	//removes the last character m3u8URL if its forwardslash
	for string(m3u8URL[len(m3u8URL)-1]) == "/" {
		m3u8URL = m3u8URL[0 : len(m3u8URL)-1]
	}
	queryParameters := ""
	m3u8Index := strings.Index(m3u8URL, ".m3u8")
	//extract query's paraments if provided.
	if m3u8Index+5 != len(m3u8URL) {
		queryParameters = m3u8URL[m3u8Index+5 : len(m3u8URL)-1]
	}
	baseUrl := m3u8URL[0:strings.LastIndex(m3u8URL, "/")]
	fmt.Printf("queryParameters:%s\nbaseUrl:%s", queryParameters, baseUrl)
	//get the list of ts files to be downloaded.
	tsList := getM3u8TsList(m3u8URL)
	if tsList == nil {
		os.Exit(1)
	}
	//makes sure there is no file with such name.
	fileIndex := 1
	baseName := fileName
	exist, err := exists(fmt.Sprintf("%s\\%s.mp4", dstFolder, fileName))
	for exist == true {
		fileName = fmt.Sprintf("%s%d", baseName, fileIndex)
		fileIndex++
		exist, err = exists(fmt.Sprintf("%s\\%s.mp4", dstFolder, fileName))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	//create a dst folder if no exists.
	exist, err = exists(dstFolder)
	for exist != true {
		os.Mkdir(dstFolder, os.ModePerm)
		exist, err = exists(dstFolder)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	//create a temp folder to store the ts files(makes sure there is no folder with such name).
	tempFolderName := fileName
	folderIndex := 1
	baseName = tempFolderName
	exist, err = exists(tempFolderName)
	for exist == true {
		tempFolderName = fmt.Sprintf("%s%d", baseName, folderIndex)
		folderIndex++
		exist, err = exists(tempFolderName)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	os.Mkdir(tempFolderName, os.ModePerm)
	fmt.Println("Downloading ts files...")
	var tsCount float64 = float64(len(tsList))
	fmt.Print("0%")
	var percents float64
	for i := 0; i < len(tsList); i++ {
		//ts file path to be created.
		tsPath := string(fmt.Sprintf("%v/%v.ts", tempFolderName, i))
		//ts url
		tsUrl := fmt.Sprintf("%v/%v", baseUrl, tsList[i])

		err = DownloadFile(tsPath, tsUrl)
		if err != nil {
			os.Exit(1)
		}
		CallClear()
		percents = (float64(i+1) / tsCount) * 100
		fmt.Printf("Downloading Progress:%.2f%%", percents)

	}
	fmt.Println("Done download ts files!\nMereging ts files...")
	mergeTsFiles(len(tsList), tempFolderName)

	fmt.Println("Saving as mp4 format...")
	saveFileAsMp4Format(tempFolderName, dstFolder, fileName)
	fmt.Printf("File saved at %s as %s.mp4\n", dstFolder, fileName)

	fmt.Println("Deleting temp files...")
	//deletes the tempFolderName folder and it's content.
	err = os.RemoveAll(tempFolderName)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Download done!, Bye")

}
