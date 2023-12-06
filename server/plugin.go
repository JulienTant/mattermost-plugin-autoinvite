package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	team *model.Team
}

// ServeHTTP demonstrates a plugin that handles HTTP requests by greeting the world.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello, world!")
}

func (p *Plugin) OnActivate() error {
	return p.loadTeam()
}

func (p *Plugin) UserHasBeenCreated(c *plugin.Context, user *model.User) {
	p.ensureUserInTeam(user.Id)
}

func (p *Plugin) UserHasLoggedIn(c *plugin.Context, user *model.User) {
	p.ensureUserInTeam(user.Id)
}

func (p *Plugin) ensureUserInTeam(userID string) {
	if p.team == nil {
		return
	}

	page := 0
	teamIds := []string{}
	for {
		teamPage, err := p.API.GetTeamMembersForUser(userID, page, 100)
		if err != nil {
			p.API.LogError("Failed to get team members for user", "user_id", userID, "page", page, "error", err.Error())
			break
		}
		for i := range teamPage {
			teamIds = append(teamIds, teamPage[i].TeamId)
		}
		if len(teamPage) < 100 {
			break
		}
		page++
	}

	alreadyInTeam := false
	for _, teamId := range teamIds {
		// if user already in team, skip
		if teamId == p.team.Id {
			alreadyInTeam = true
			break
		}
	}

	if !alreadyInTeam {
		_, err := p.API.CreateTeamMember(p.team.Id, userID)
		if err != nil {
			p.API.LogError("Failed to add user to team", "user_id", userID, "team_id", p.team.Id, "error", err.Error())
		}
	}
}

func (p *Plugin) loadTeam() error {
	if p.configuration.TeamName == "" {
		p.team = nil
		return nil
	}

	team, err := p.API.GetTeamByName(p.configuration.TeamName)
	if err != nil {
		return err
	}

	p.team = team
	return nil
}

// See https://developers.mattermost.com/extend/plugins/server/reference/
