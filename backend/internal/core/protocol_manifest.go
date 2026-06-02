package core

func ProtocolManifest() ProtocolManifestResponse {
	return ProtocolManifestResponse{
		ProtocolVersion: "mergeos.protocol.manifest.v1",
		Kind:            "protocol_manifest",
		Schemas: []ProtocolManifestSchema{
			{
				Version:     "mergeos.task.v1",
				Kind:        "task",
				SchemaURL:   "https://mergeos.shop/protocol/task.v1.schema.json",
				Description: "Claimable bounty task with reward, worker lane, acceptance criteria, and evidence requirements.",
			},
			{
				Version:     "mergeos.agent.v1",
				Kind:        "agent",
				SchemaURL:   "https://mergeos.shop/protocol/agent.v1.schema.json",
				Description: "AI agent lane with supported actions, capabilities, active bounty demand, and open task references.",
			},
			{
				Version:     "mergeos.workflow.v1",
				Kind:        "workflow",
				SchemaURL:   "https://mergeos.shop/protocol/workflow.v1.schema.json",
				Description: "Project workflow graph with progress, current AI workflow step, task nodes, dependency edges, readiness, and release status.",
			},
			{
				Version:     "mergeos.event.v1",
				Kind:        "event",
				SchemaURL:   "https://mergeos.shop/protocol/event.v1.schema.json",
				Description: "Realtime project, task, PR, deployment, ledger, and AI agent event envelope.",
			},
			{
				Version:     "mergeos.scan.v1",
				Kind:        "repository_scan",
				SchemaURL:   "https://mergeos.shop/protocol/scan.v1.schema.json",
				Description: "Repository scan findings for security, dependency, quality, and technical debt signals.",
			},
		},
		Endpoints: []ProtocolManifestEndpoint{
			{
				Method:      "GET",
				Path:        "/api/public/protocol",
				Auth:        "none",
				Description: "Protocol manifest and endpoint discovery for external agents and integrations.",
			},
			{
				Method:      "GET",
				Path:        "/api/public/protocol/tasks",
				Protocol:    "mergeos.task.v1",
				Auth:        "none",
				Description: "Public open bounty tasks as protocol documents.",
			},
			{
				Method:      "GET",
				Path:        "/api/public/protocol/agents",
				Protocol:    "mergeos.agent.v1",
				Auth:        "none",
				Description: "Public AI agent lanes as protocol documents.",
			},
			{
				Method:      "GET",
				Path:        "/api/public/protocol/events",
				Protocol:    "mergeos.event.v1",
				Auth:        "none",
				Description: "Public live-feed events as protocol documents.",
			},
			{
				Method:      "GET",
				Path:        "/api/projects/{id}/protocol/workflow",
				Protocol:    "mergeos.workflow.v1",
				Auth:        "project",
				Description: "Authenticated project workflow graph protocol document.",
			},
			{
				Method:      "GET",
				Path:        "/api/projects/{id}/protocol/scan",
				Protocol:    "mergeos.scan.v1",
				Auth:        "project",
				Description: "Authenticated repository scan protocol document.",
			},
		},
	}
}
