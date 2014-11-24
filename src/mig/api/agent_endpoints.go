// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"fmt"
	"github.com/jvehent/cljs"
	"mig"
	migdb "mig/database"
	"net/http"
	"strconv"
)

func getAgent(respWriter http.ResponseWriter, request *http.Request) {
	var err error
	opid := getOpID(request)
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving getAgentsDashboard()"}.Debug()
	}()
	agentID, err := strconv.ParseFloat(request.URL.Query()["agentid"][0], 64)
	if err != nil {
		err = fmt.Errorf("Wrong parameters 'agentid': '%v'", err)
		panic(err)
	}

	// retrieve the command
	var agt mig.Agent
	if agentID > 0 {
		agt, err = ctx.DB.AgentByID(agentID)
		if err != nil {
			if fmt.Sprintf("%v", err) == "Error while retrieving agent: 'sql: no rows in result set'" {
				// not found, return 404
				resource.SetError(cljs.Error{
					Code:    fmt.Sprintf("%.0f", opid),
					Message: fmt.Sprintf("Agent ID '%.0f' not found", agentID)})
				respond(404, resource, respWriter, request)
				return
			} else {
				panic(err)
			}
		}
	} else {
		// bad request, return 400
		resource.SetError(cljs.Error{
			Code:    fmt.Sprintf("%.0f", opid),
			Message: fmt.Sprintf("Invalid Agent ID '%.0f'", agentID)})
		respond(400, resource, respWriter, request)
		return
	}
	// store the results in the resource
	agentItem, err := agentToItem(agt)
	if err != nil {
		panic(err)
	}
	resource.AddItem(agentItem)
	respond(200, resource, respWriter, request)
}

// agentToItem receives an agent and returns an Item in Collection+JSON
func agentToItem(agt mig.Agent) (item cljs.Item, err error) {
	item.Href = fmt.Sprintf("%s/agent?agentid=%.0f", ctx.Server.BaseURL, agt.ID)
	item.Data = []cljs.Data{
		{Name: "agent", Value: agt},
	}
	return
}

// agentsSumToItem receives an AgentsSum and returns an Item
// in the Collection+JSON format
func agentsSummaryToItem(sum []migdb.AgentsSum, count, double, disappeared float64, ctx Context) (item cljs.Item, err error) {
	item.Href = fmt.Sprintf("%s/dashboard", ctx.Server.BaseURL)
	var total float64 = 0
	for _, asum := range sum {
		total += asum.Count
	}
	item.Data = []cljs.Data{
		{Name: "active agents", Value: total},
		{Name: "agents versions count", Value: sum},
		{Name: "agents started in the last 24 hours", Value: count},
		{Name: "endpoints running 2 or more agents", Value: double},
		{Name: "endpoints that have disappeared over last 7 days", Value: disappeared},
	}
	return
}