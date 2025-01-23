package getpocket

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/juev/getpocket/internal/client"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type Pocket struct {
	ConsumerKey string `json:"consumer_key"`
	AccessToken string `json:"access_token"`
	State       string `json:"state"`
	DetailType  string `json:"detailType"`
	Count       int    `json:"count"`
	Offset      int    `json:"offset"`
	Total       int    `json:"total"`
	body        string
}

type PocketItem struct {
	ID                     string            `json:"item_id"`
	Favorite               string            `json:"favorite"`
	Status                 string            `json:"status"`
	TimeAdded              string            `json:"time_added"`
	TimeUpdated            string            `json:"time_updated"`
	TimeRead               string            `json:"time_read"`
	TimeFavorited          string            `json:"time_favorited"`
	SortID                 int               `json:"sort_id"`
	Tags                   map[string]string `json:"tags"`
	TopImageURL            string            `json:"top_image_url"`
	ResolvedID             string            `json:"resolved_id"`
	GivenURL               string            `json:"given_url"`
	GivenTitle             string            `json:"given_title"`
	ResolvedTitle          string            `json:"resolved_title"`
	ResolvedURL            string            `json:"resolved_url"`
	Excerpt                string            `json:"excerpt"`
	IsArticle              string            `json:"is_article"`
	IsIndex                string            `json:"is_index"`
	HasVideo               string            `json:"has_video"`
	HasImage               string            `json:"has_image"`
	WordCount              string            `json:"word_count"`
	Lang                   string            `json:"lang"`
	TimeToRead             int               `json:"time_to_read"`
	ListenDurationEstimate int               `json:"listen_duration_estimate"`
}

const (
	endpoint            = "https://getpocket.com/v3/get"
	pocketCount         = 30
	pocketTotal         = 1
	pocketDefaultOffset = 0
	pocketState         = "unread"
	pocketDetailType    = "simple"
)

var (
	ErrSomethingWentWrong = errors.New("Something Went Wrong")
)

// New creates a new pocket instance with the given consumer key and access token.
func New(consumerKey, accessToken string) (*Pocket, error) {
	p := &Pocket{
		ConsumerKey: consumerKey,
		AccessToken: accessToken,
		State:       pocketState,
		DetailType:  pocketDetailType,
		Count:       pocketCount,
		Offset:      pocketDefaultOffset,
		Total:       pocketTotal,
	}

	body, _ := json.Marshal(p)
	p.body = string(body)

	return p, nil
}

func (p *Pocket) Retrive(since int64) ([]PocketItem, int64, error) {
	var (
		newSince int64
		result   []PocketItem
		err      error
	)

	offset := pocketDefaultOffset
	count := pocketCount
	for count > 0 {
		var items []PocketItem
		items, newSince, err = p.request(since, offset)
		if err != nil {
			return nil, since, err
		}
		count = len(items)
		result = append(result, items...)
		offset += pocketCount
	}

	return result, newSince, nil
}

func (p *Pocket) request(since int64, offset int) ([]PocketItem, int64, error) {
	request, _ := http.NewRequest(http.MethodPost, endpoint, nil)
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("X-Accept", "application/json")

	body := p.body
	body, _ = sjson.Set(body, "since", since)
	body, _ = sjson.Set(body, "offset", offset)
	request.Body = io.NopCloser(strings.NewReader(body))
	response, err := client.Request(request)
	if err != nil {
		return nil, since, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, since, fmt.Errorf("got response %d; X-Error=[%s]", response.StatusCode, response.Header.Get("X-Error"))
	}

	bodyString := response.Body
	if e := gjson.Get(bodyString, "error").String(); e != "" {
		return nil, since, ErrSomethingWentWrong
	}

	// Update since
	newSince := gjson.Get(bodyString, "since").Int()

	if gjson.Get(bodyString, "status").Int() == 2 {
		return nil, newSince, nil
	}

	list := gjson.Get(bodyString, "list").Map()
	var result []PocketItem
	for k := range list {
		value := list[k].String()

		var pp PocketItem
		if err := json.Unmarshal([]byte(value), &pp); err != nil {
			return nil, since, err
		}
		result = append(result, pp)
	}

	return result, newSince, nil
}
