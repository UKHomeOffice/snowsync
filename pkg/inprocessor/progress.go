package inprocessor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/UKHomeOffice/snowsync/pkg/client"
)

type resolution struct {
	Com comment `json:"comment,omitempty"`
}

type add struct {
	Body string `json:"body"`
}

type comment []struct {
	Action add `json:"add,omitempty"`
}

type transition struct {
	ID string `json:"id,omitempty"`
}

func (p *Processor) progress(i Incident) (string, error) {

	fmt.Printf("debug msg into progress %+v\n", i)

	user, pass, base, err := getEnv()
	if err != nil {
		return "", fmt.Errorf("environment error: %v", err)
	}

	surl, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("could not form JSD URL: %v", err)
	}

	// create client and request
	c := &client.Client{
		BaseURL:    surl,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}

	// only allowing Investigating and Resolved at MVP stage
	var t string
	switch i.Status {
	case "":
		fmt.Printf("ignoring blank status %v", i.Status)
		return i.ExtID, nil
	case "1":
		fmt.Printf("ignoring status %v", i.Status)
		return i.ExtID, nil
	case "10100":
		t = "11"
	case "3":
		t = "71"
	default:
		return "", fmt.Errorf("unexpected ticket status: %v", i.Status)
	}

	// add final resolution comment

	var rc resolution
	var co comment

	co = make(comment, 0)
	co = append(co, struct {
		Action add "json:\"add,omitempty\""
	}{add{Body: i.Resolution}})
	rc.Com = co

	v := Values{
		Resolution: &rc,
		Transition: &transition{ID: t},
	}

	path, err := url.Parse("/rest/api/2/issue/" + i.ExtID + "/transitions")
	if err != nil {
		return "", fmt.Errorf("could not form JSD URL: %v", err)
	}
	out, err := json.Marshal(&v)
	if err != nil {
		return "", fmt.Errorf("could marshal JSD payload: %v", err)
	}
	req, err := c.NewRequest(path.Path, "POST", user, pass, out)
	if err != nil {
		return "", fmt.Errorf("could not make request: %v", err)
	}
	// make HTTP request to JSD
	res, err := c.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not call JSD: %v", err)
	}
	defer res.Body.Close()

	return i.ExtID, nil
}
