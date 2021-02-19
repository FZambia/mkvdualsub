package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	formEndpoint   = "https://pas-bien.net/2srt2ass/"
	requestTimeout = 5 * time.Second
)

func getAssFile(client *http.Client, values map[string]io.Reader, outFile string) error {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	for key, r := range values {
		var fw io.Writer
		if x, ok := r.(*os.File); ok {
			var err error
			if fw, err = w.CreateFormFile(key, x.Name()); err != nil {
				return err
			}
		} else {
			var err error
			if fw, err = w.CreateFormField(key); err != nil {
				return err
			}
		}
		if _, err := io.Copy(fw, r); err != nil {
			return err
		}
	}
	_ = w.Close()

	req, err := http.NewRequest("POST", formEndpoint, &b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status code: %s", resp.Status)
	}

	out, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, resp.Body)
	return err
}

func mustOpen(f string) *os.File {
	r, err := os.Open(f)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return r
}

func mustGetPath(program string) string {
	path, err := exec.LookPath(program)
	if err != nil {
		fmt.Printf("could not find %s", program)
		os.Exit(1)
	}
	return path
}

func exitWithErr(err error) {
	fmt.Println(err)
	os.Exit(1)
}

type subtitleInfo struct {
	Track int
	Info  string
}

func extractSubtitleInfo(pathMerge string, file string) ([]subtitleInfo, error) {
	execCmd := exec.Command(pathMerge, "-i", file)
	out, err := execCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error calling mkvmerge: %v", err)
	}
	var subtitles []subtitleInfo
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "subtitles") {
			// Track ID 5: subtitles (SubRip/SRT)
			trimmed := strings.TrimPrefix(line, "Track ID ")
			colonIndex := strings.Index(trimmed, ":")
			if colonIndex <= 0 {
				return nil, fmt.Errorf("malformed subtitle line: %s", line)
			}
			trackNumber, err := strconv.Atoi(trimmed[:colonIndex])
			if err != nil {
				return nil, fmt.Errorf("malformed subtitle number: %s", line)
			}
			subtitles = append(subtitles, subtitleInfo{
				Track: trackNumber,
				Info:  line,
			})
		}
	}
	if scanner.Err() != nil {
		return nil, scanner.Err()
	}
	return subtitles, nil
}

func mustValidTrackNumber(num int, subtitles []subtitleInfo) {
	for _, sub := range subtitles {
		if sub.Track == num {
			return
		}
	}
	exitWithErr(fmt.Errorf("invalid number of subtitle track: (%d)", num))
}

func main() {
	pathMerge := mustGetPath("mkvmerge")
	pathExtract := mustGetPath("mkvextract")

	var cmdInfo = &cobra.Command{
		Use:   "info [file]",
		Short: "Information about file subtitles",
		Long:  `Information about file subtitles`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			subtitles, err := extractSubtitleInfo(pathMerge, args[0])
			if err != nil {
				exitWithErr(err)
			}
			for _, sub := range subtitles {
				fmt.Printf("%v\n", sub.Info)
			}
		},
	}

	var topTrack int
	var bottomTrack int

	var cmdJoin = &cobra.Command{
		Use:   "join [file]",
		Short: "Join subtitles",
		Long:  `Join subtitles`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			filePath := args[0]
			subtitles, err := extractSubtitleInfo(pathMerge, filePath)
			if err != nil {
				exitWithErr(err)
			}

			if len(subtitles) < 2 {
				exitWithErr(fmt.Errorf("number of available subtitles less than 2 (%d)", len(subtitles)))
			}

			if topTrack < 0 && bottomTrack < 0 {
				topTrack = subtitles[0].Track
				bottomTrack = subtitles[1].Track
			} else if topTrack < 0 {
				mustValidTrackNumber(bottomTrack, subtitles)
				for _, sub := range subtitles {
					if sub.Track != bottomTrack {
						topTrack = sub.Track
						break
					}
				}
			} else if bottomTrack < 0 {
				mustValidTrackNumber(topTrack, subtitles)
				for _, sub := range subtitles {
					if sub.Track != topTrack {
						bottomTrack = sub.Track
						break
					}
				}
			} else {
				mustValidTrackNumber(bottomTrack, subtitles)
				mustValidTrackNumber(topTrack, subtitles)
			}

			topFile := filepath.Join(os.TempDir(), "mkvdualsub_top.srt")
			track := fmt.Sprintf("%d:%s", topTrack, topFile)
			execCmd := exec.Command(pathExtract, "tracks", filePath, track)
			_, err = execCmd.Output()
			if err != nil {
				exitWithErr(err)
			}
			defer func() { _ = os.Remove(topFile) }()

			bottomFile := filepath.Join(os.TempDir(), "mkvdualsub_bottom.srt")
			track = fmt.Sprintf("%d:%s", bottomTrack, bottomFile)
			execCmd = exec.Command(pathExtract, "tracks", filePath, track)
			_, err = execCmd.Output()
			if err != nil {
				exitWithErr(err)
			}
			defer func() { _ = os.Remove(bottomFile) }()

			client := &http.Client{Timeout: requestTimeout}

			topFileReader := mustOpen(topFile)
			defer func() { _ = topFileReader.Close() }()

			bottomFileReader := mustOpen(bottomFile)
			defer func() { _ = bottomFileReader.Close() }()

			form := map[string]io.Reader{
				"top":      topFileReader,
				"bot":      bottomFileReader,
				"send":     strings.NewReader("yes"),
				"fontname": strings.NewReader("Arial"),
				"fontsize": strings.NewReader("16"),
				"topColor": strings.NewReader("#FFFFF9"),
				"botColor": strings.NewReader("#F9FFF9"),
			}

			outFile := args[0] + ".ass"
			if _, err := os.Stat(outFile); err == nil {
				exitWithErr(fmt.Errorf("%s already exists", outFile))
			}

			err = getAssFile(client, form, outFile)
			if err != nil {
				exitWithErr(err)
			}
		},
	}

	cmdJoin.Flags().IntVarP(&bottomTrack, "bottom", "b", -1, "top subtitle track number")
	cmdJoin.Flags().IntVarP(&topTrack, "top", "t", -1, "bottom subtitle track number")

	var rootCmd = &cobra.Command{Use: "mkvdualsub"}
	rootCmd.AddCommand(cmdInfo, cmdJoin)
	_ = rootCmd.Execute()
}
