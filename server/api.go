package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

func (p *Plugin) InitAPI() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/triggerBuild", p.handleBuildTrigger).Methods("POST")
	r.HandleFunc("/createJob", p.handleJobCreation).Methods("POST")
	return r
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()

	if err := p.IsValid(config); err != nil {
		http.Error(w, "This plugin is not configured.", http.StatusNotImplemented)
		return
	}
	p.API.LogDebug("New request:", "Host", r.Host, "RequestURI", r.RequestURI, "Method", r.Method)
	p.router.ServeHTTP(w, r)
}

func (p *Plugin) handleBuildTrigger(w http.ResponseWriter, r *http.Request) {
	jobName := r.FormValue("jobName")
	decodedJobName, _ := url.QueryUnescape(jobName)

	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	body, _ := ioutil.ReadAll(r.Body)
	bodyString := string(body)

	request := model.SubmitDialogRequestFromJson(strings.NewReader(bodyString))
	if request == nil {
		p.API.LogError("failed to decode request")
		return
	}

	parameters := make(map[string]string)
	for k, v := range request.Submission {
		parameters[k] = v.(string)
	}

	build, err := p.triggerJenkinsJob(userID, request.ChannelId, jobName, parameters)
	if err != nil {
		p.API.LogError("Error triggering build", "job_name", jobName, "err", err.Error())
		return
	}
	p.createPost(userID, request.ChannelId, fmt.Sprintf("Job '%s' - #%d has been started\nBuild URL : %s", decodedJobName, build.GetBuildNumber(), build.GetUrl()))
}

func (p *Plugin) handleJobCreation(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	body, _ := ioutil.ReadAll(r.Body)
	bodyString := string(body)

	request := model.SubmitDialogRequestFromJson(strings.NewReader(bodyString))
	if request == nil {
		p.API.LogError("failed to decode request")
		return
	}

	jobInputs := make(map[string]string)
	for k, v := range request.Submission {
		jobInputs[k] = v.(string)
	}
	p.sendJobCreateRequest(userID, request.ChannelId, jobInputs)
}
