package weatherinfo

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"bufio"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	//~ "log"
)

var (
	KanaName = map[string]string{
		"快晴": "ｶｲｾｲ", "晴れ": "ﾊﾚ", "薄曇り": "ｳｽｸﾓﾘ", "曇り": "ｸﾓﾘ",
		"煙霧": "ｴﾝﾑ", "砂じん嵐": "ｻｼﾞﾝｱﾗｼ", "地ふぶき": "ｼﾞﾌﾌﾞｷ", "霧": "ｷﾘ",
		"霧雨": "ｷﾘｻﾒ", "雨": "ｱﾒ", "みぞれ": "ﾐｿﾞﾚ", "雪": "ﾕｷ",
		"強雨":  "ｷｮｳｳ",
		"あられ": "ｱﾗﾚ", "ひょう": "ﾋｮｳ", "雷": "ｶﾐﾅﾘ",
		"時々": "ﾄｷﾄﾞｷ", "一時": "ｲﾁｼﾞ", "のち": "ﾉﾁ", "続く": "ﾂﾂﾞｸ",
		"後": "ﾉﾁ", "今日": "ｷｮｳ ", "今夜": "ｺﾝﾔ ", "明日": "ｱｼﾀ ", "明後日": "ｱｻｯﾃ ",
		"大雨": "ｵｵｱﾒ", "洪水": "ｺｳｽﾞｲ", "強風": "ｷｮｳﾌｳ",
		"風雪": "ﾌｳｾﾂ", "大雪": "ｵｵﾕｷ", "波浪": "ﾊﾛｳ",
		"高潮": "ﾀｶｼｵ", "融雪": "ﾕｳｾﾂ",
		"濃霧": "ﾉｳﾑ", "乾燥": "ｶﾝｿｳ", "なだれ": "ﾅﾀﾞﾚ",
		"低温": "ﾃｲｵﾝ", "霜": "ｼﾓ", "着氷": "ﾁｬｸﾋｮｳ",
		"着雪": "ﾁｬｸｾﾂ", "暴風": "ﾎﾞｳﾌｳ", "暴風雪": "ﾎﾞｳﾌｳｾﾂ",
		"特別警報": "ﾄｸﾍﾞﾂｹｲﾎｳ", "警報": "ｹｲﾎｳ", "注意報": "ﾁｭｳｲﾎｳ",
		"解除": "ｶｲｼﾞｮ", "無し": "ﾅｼ", "警報・注意報": "ｹｲﾎｳ･ﾁｭｳｲﾎｳ", "": ""}

	timezone = []string{"ﾐﾒｲ", "ｱｹｶﾞﾀ", "ｱｻ", "ﾋﾙﾏｴ",
		"ﾋﾙｽｷﾞ", "ﾕｳｶﾞﾀ", "ｺﾝﾊﾞﾝ", "ﾖﾙｵｿｸ",
		"ｱｽ ﾐﾒｲ", "ｱｽ ｱｹｶﾞﾀ", "ｱｽ ｱｻ", "ｱｽ ﾋﾙﾏｴ",
		"ｱｽ ﾋﾙｽｷﾞ", "ｱｽ ﾕｳｶﾞﾀ", "ｱｽ ﾖﾙ", "ｱｽ ﾖﾙｵｿｸ"}

	outputdir string = "/run/user/1000/weatherinfo"
	urllist          = make(map[string]string)
)

type Forecast struct {
	Weather       string
	Termperature  int16
	Humidity      int16
	Precipitation float64
	Direction     string
	Speed         int16
}

type WarnInfo struct {
	Label     string
	AlarmType string
}

type WeeklyInfo struct {
	Date        time.Time
	Weather     string
	Temperature int16
	COR         int16 // Chance of Rainfall
}

type Weatherinfo3 struct {
	workingdir string
	outputfile string

	LocName  string
	ForeData [16]Forecast
	Warning  []WarnInfo
	Weekly   [6]WeeklyInfo
}

func New() *Weatherinfo3 {
	return &Weatherinfo3{
		workingdir: outputdir,
		outputfile: "yjw.html",
	}
}

func SetWorkingDir(workingdir string) {
	outputdir = workingdir
}

func (w *Weatherinfo3) SetWorkingDir(workingdir string) {
	w.workingdir = workingdir
}

func (w *Weatherinfo3) GetWorkingDir() string {
	return w.workingdir
}

func (w *Weatherinfo3) SetOutputFile(s string) {
	w.outputfile = s
}

func (w *Weatherinfo3) GetOutputFile() string {
	return w.outputfile
}

func (w *Weatherinfo3) GetWeatherInfo(url string, label string) error {
	if err := os.MkdirAll(w.workingdir, 0755); err != nil {
		return err
	}
	output := filepath.Join(w.workingdir, w.outputfile)
	currentTime := time.Now()
	if err := Download(output, url); err != nil {
		return err
	}

	f, err := os.Open(output)
	if err != nil {
		return err
	}
	defer f.Close()

	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		return err
	}

	//
	//  地名
	//
	w.LocName = label

	//
	//  警報・注意報
	//
	doc.Find("div#wrnrpt").Children().Find("dt").Each(func(i int, dt *goquery.Selection) {
		if dt.Text() != "発表なし" {
			dd := dt.Next()
			a := WarnInfo{Label: dt.Text(), AlarmType: dd.Text()}
			w.Warning = append(w.Warning, a)
		} else {
			a := WarnInfo{Label:"警報・注意報", AlarmType:"無し"}
			w.Warning = append(w.Warning, a)
		}
	})
	//
	//  天気予報
	//
	taginfo := map[string]string{"div#yjw_pinpoint_today": "今日",
		"div#yjw_pinpoint_tomorrow": "明日",
		"div#yjw_week":              "週間予報"}

	rep := regexp.MustCompile(`<.+?>`)
	for day, label := range taginfo {
		tpx := 0
		tpy := -1
		var buf [9][6]string

		doc.Find(day).Each(func(i int, s *goquery.Selection) {
			s.Children().Find("tr").Each(func(i1 int, s0 *goquery.Selection) {
				s0.Children().Each(func(i2 int, s1 *goquery.Selection) {
					s, _ := s1.Html()
					//
					// 本文が複数行から構成される場合は一旦切り離し、
					// 余計な空白やタグを落としてから再結合する
					//
					var sTmp string
					for _, i := range strings.Split(s, "<br/>") {
						item := strings.TrimRight(
							strings.TrimLeft(
								strings.ReplaceAll(
									rep.ReplaceAllString(i, ""),
									"\n", ""),
								" "),
							" ")
						if sTmp != "" {
							sTmp += " "
						}
						sTmp += item
					}

					if i2 == 0 {
						tpx = 0
						tpy++
					} else {
						buf[tpx][tpy] = sTmp
						tpx++
					}
				})
			})
		})

		ofset := 0
		switch label {
		case "明日":
			ofset = 8
			fallthrough
		case "今日":
			for x := 0; x < 8; x++ {
				var tmp int
				w.ForeData[ofset+x].Weather = buf[x][1]
				tmp, _ = strconv.Atoi(buf[x][2])
				w.ForeData[ofset+x].Termperature = int16(tmp)
				tmp, _ = strconv.Atoi(buf[x][3])
				w.ForeData[ofset+x].Humidity = int16(tmp)
				w.ForeData[ofset+x].Precipitation, _ = strconv.ParseFloat(buf[x][4], 64)
				ss := strings.Split(buf[x][5], " ")
				w.ForeData[ofset+x].Direction = ss[0]
				tmp, _ = strconv.Atoi(ss[1])
				w.ForeData[ofset+x].Speed = int16(tmp)
			}

		case "週間予報":
			var (
				tmp int
				t2  time.Time
			)
			t2 = currentTime
			d2, _ := time.ParseDuration("24h00m")
			t2 = t2.Add(d2)
			for x := 0; x < 6; x++ {
				t2 = t2.Add(d2)
				w.Weekly[x].Date = t2
				w.Weekly[x].Weather = buf[x][1]
				tmp, _ = strconv.Atoi(buf[x][2])
				w.Weekly[x].Temperature = int16(tmp)
				tmp, _ = strconv.Atoi(buf[x][3])
				w.Weekly[x].COR = int16(tmp) // Chance of Rainfall
			}
		}
	}
	return nil
}

func (w *Weatherinfo3) GetHoursLaterInfo(after int) (*string, *Forecast) {
	// 現在時刻から指定時間後の予報を返す。データは自動更新されない。
	// 24時間以上先を指定した場合は nil,nil を返す
	d1 := time.Now().Hour() + after

	if d1 > 23+24 {
		return nil, nil
	}
	d1 /= 3

	return &timezone[d1], &w.ForeData[d1]
}

func Download(path string, url string) error {
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, res.Body)
	return err
}

// 予報対象地域の名称から該当するURLを得る
// 地域名をkeyにしたmapを返す
// 調べる地域名称が不完全な場合、それが含まれる全ての地域について返す
// mapは使いまわす。（この関数を呼び出すたびに新規作成される訳ではない）
func ForecastUrlTargetArea(cityname string) (*map[string]string, error) {
	var err error
	if err = os.MkdirAll(outputdir, 0755); err != nil {
		return nil, err
	}

	address := fmt.Sprintf("https://weather.yahoo.co.jp/weather/search/?p=%s", url.PathEscape(cityname))
	outputfile := filepath.Join(outputdir, "list.html")

	titlefile := filepath.Join(outputdir, "title.txt")
	cached := false
	tf, err := os.Open(titlefile)
	if err == nil { // ファイルが存在しない場合はそのまま進める
		defer tf.Close()

		scanner := bufio.NewScanner(tf)
		for scanner.Scan() {
			s := scanner.Text()
			if strings.Contains(s, fmt.Sprintf("%sの", cityname)) {
				cached = true
				break
			}
		}
	}

	if cached == false {
		if err = Download(outputfile, address); err != nil {
			return nil, err
		}
	}

	f, err := os.Open(outputfile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		return nil, err
	}

	// ダウンロード済みであることを記録しておく
	v, err := doc.Find("title").Html()
	if err != nil {
		return nil, err
	}
	wf, err := os.Create(titlefile)
	if err != nil {
		return nil, err
	}
	defer wf.Close()
	_, err = wf.WriteString(v)
	if err != nil {
		return nil, err
	}

	doc.Find("div > table > tbody > tr > td > a").Each(func(i int, s *goquery.Selection) {
		url, _ := s.Attr("href")
		label := s.Text()
		if strings.Index(url, "https:") != -1 {
			if urllist[label] == "" {
				urllist[label] = url
			}
		}
	})
	return &urllist, err
}

//~ 未明: 0時～3時			明け方: 3時～6時		朝: 6時～9時
//~ 昼前: 9時～12時		昼過ぎ: 12時～15時		夕方: 15時～18時
//~ 夜のはじめ頃: 18時～21時	夜遅く: 21時～24時

