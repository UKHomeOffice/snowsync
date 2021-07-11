package in

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/UKHomeOffice/snowsync/pkg/caller"
)

// Values make up the JSD payload
type Values struct {
	Comment     string      `json:"comment,omitempty"`
	Description string      `json:"description,omitempty"`
	Summary     string      `json:"summary,omitempty"`
	Priority    *priority   `json:"priority,omitempty"`
	Transition  *transition `json:"transition,omitempty"`
}

type priority struct {
	Name string `json:"name,omitempty"`
}

type transition struct {
	ID string `json:"id,omitempty"`
}

func transformCreate(p *Incident) (map[string]interface{}, error) {

	dat := make(map[string]interface{})
	dat["serviceDeskId"] = "1"
	dat["requestTypeId"] = "14"

	// priority sync is out of scope for MVP so hardcoding
	pri := priority{
		Name: "P4 - General request",
	}

	v := Values{
		Priority: &pri,
		Summary:  p.Summary,
		// FIXME: don't append initial comment to description
		Description: fmt.Sprintf("Incident %v raised on ServiceNow by %v with priority %v.\n Description: %v.\n Initial comment (%v %v): %v %v",
			p.IntID, p.Reporter, p.Priority, p.Description, p.CommentID, p.IntCommentID, p.Comment, p.IntComment),
	}

	dat["requestFieldValues"] = v
	return dat, nil

}

func createIncident(b []byte) (string, error) {

	user, pass, base, err := getEnv()
	if err != nil {
		return "", fmt.Errorf("environment error: %v", err)
	}

	surl, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("could not form JSD URL: %v", err)
	}

	// create client and request
	c := &caller.Client{
		BaseURL:    surl,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}

	//FIXME: temp URL
	req, err := c.NewRequest("/tabf69k/rest/servicedeskapi/request/", "POST", user, pass, b)
	if err != nil {
		return "", fmt.Errorf("could not make request: %v", err)
	}

	// make HTTP request to JSD
	res, err := c.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not call JSD: %v", err)
	}
	defer res.Body.Close()

	// read HTTP response
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("could not read JSD response body %v", err)
	}

	// FIXME
	//fmt.Printf("sent request, JSD replied with: %v", string(body))

	// dynamically decode response and check for JSD assigned identifier
	var dat map[string]interface{}
	err = json.Unmarshal(body, &dat)
	if err != nil {
		return "", fmt.Errorf("could not decode JSD response: %v", err)
	}

	eid, ok := dat["issueKey"].(string)
	if !ok && eid == "" {
		return "", fmt.Errorf("could not find an identifier in JSD response")
	}

	fmt.Printf("JSD returned an identifier: %v", eid)
	return eid, nil

}

func (p *Processor) create(in *Incident) (string, error) {

	v, err := transformCreate(in)
	if err != nil {
		return "", fmt.Errorf("could not transform creator payload: %v", err)
	}

	new, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("could not marshal creator payload: %v", err)
	}

	out, err := createIncident(new)
	if err != nil {
		return "", fmt.Errorf("could not invoke a create call: %v", err)
	}

	return out, nil
}

func transformUpdate(p *Incident) (map[string]interface{}, error) {

	dat := make(map[string]interface{})

	dat["external_identifier"] = p.ExtID
	dat["body"] = fmt.Sprintf("Comment added on ServiceNow (%v): %v", p.CommentID, p.Comment)

	return dat, nil
}

func updateIncident(b []byte) (string, error) {

	user, pass, base, err := getEnv()
	if err != nil {
		return "", fmt.Errorf("environment error: %v", err)
	}

	surl, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("could not form JSD URL: %v", err)
	}

	// create client and request
	c := &caller.Client{
		BaseURL:    surl,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}

	// remove the need for this switcheroo
	var dat map[string]interface{}
	err = json.Unmarshal(b, &dat)
	if err != nil {
		return "", fmt.Errorf("could not decode payload to get external id: %v", err)
	}

	eid, ok := dat["external_identifier"].(string)
	if ok {
		delete(dat, "external_identifier")
		//FIXME: temp URL
		path, err := url.Parse("/tabf69k/rest/api/2/issue/" + eid + "/comment")
		if err != nil {
			return "", fmt.Errorf("could not form JSD URL: %v", err)
		}
		out, err := json.Marshal(&dat)
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

		// read HTTP response
		_, err = ioutil.ReadAll(res.Body)
		if err != nil {
			return "", fmt.Errorf("could not read JSD response body %v", err)
		}

		// FIXME
		//fmt.Printf("sent request, JSD replied with: %v", string(body))
		return eid, nil
	}
	return "", fmt.Errorf("no identifier in payload")
}

func (p *Processor) update(pay *Incident) (string, error) {

	v, err := transformUpdate(pay)
	if err != nil {
		return "", fmt.Errorf("could not transform creator payload: %v", err)
	}

	upd, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("could not marshal updater payload: %v", err)
	}

	out, err := updateIncident(upd)
	if err != nil {
		return "", fmt.Errorf("could not invoke a create call: %v", err)
	}

	return out, nil
}

func (p *Processor) progress(pay *Incident) (string, error) {

	user, pass, base, err := getEnv()
	if err != nil {
		return "", fmt.Errorf("environment error: %v", err)
	}

	surl, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("could not form JSD URL: %v", err)
	}

	// create client and request
	c := &caller.Client{
		BaseURL:    surl,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}

	// only allowing Investtigating and Resolved at MVP stage
	var t string
	switch pay.Status {
	case "":
		fmt.Printf("\nignoring blank status %v\n", pay.Status)
		return pay.ExtID, nil
	case "1":
		fmt.Printf("\nignoring status %v\n", pay.Status)
		return pay.ExtID, nil
	case "10100":
		t = "11"
	case "3":
		t = "71"
	default:
		return "", fmt.Errorf("\nunexpected ticket status: %v", pay.Status)
	}

	v := Values{
		Transition: &transition{ID: t},
	}

	//FIXME: temp URL
	path, err := url.Parse("/tabf69k/rest/api/2/issue/" + pay.ExtID + "/transitions")
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

	return pay.ExtID, nil
}
