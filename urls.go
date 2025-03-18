package rtve

var urlMap = map[string]*Show{
	"telediario-2": {
		ID:    "135930",
		URL:   "https://www.rtve.es/play/videos/modulos/capitulos/135930/?page=%d",
		Regex: `https://www\.rtve\.es/play/videos/telediario-2/.*/`,
	},
	"telediario-1": {
		URL:   "https://www.rtve.es/play/videos/modulos/capitulos/45030/?page=%d",
		ID:    "45030",
		Regex: `https://www\.rtve\.es/play/videos/telediario-1/.*/`,
	},
	"telediario-matinal": {
		URL:   "https://www.rtve.es/play/videos/modulos/capitulos/135931/?page=%d",
		ID:    "135931",
		Regex: `https://www\.rtve\.es/play/videos/telediario-matinal/.*/`,
	},
	"informe-semanal": {
		URL:   "https://www.rtve.es/play/videos/modulos/capitulos/1631/?page=%d",
		ID:    "1631",
		Regex: `https://www\.rtve\.es/play/videos/informe\-semanal/.*/`,
	},
}

const ApiURL = "https://api2.rtve.es/api/videos/%s.json"
const SubsURL = "https://api2.rtve.es/api/videos/%s/subtitulos.json"

type Show struct {
	ID    string
	URL   string
	Regex string
}

func ShowMap(name string) *Show {
	return (urlMap[name])
}

func ListShows() []string {
	var shows []string
	for k, _ := range urlMap {
		shows = append(shows, k)
	}
	return shows
}
