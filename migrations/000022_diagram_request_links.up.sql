CREATE TABLE diagram_request_links (
    diagram_id UUID NOT NULL REFERENCES workspace_diagrams(id) ON DELETE CASCADE,
    request_id UUID NOT NULL REFERENCES requests(id) ON DELETE CASCADE,
    PRIMARY KEY (diagram_id, request_id)
);

CREATE INDEX idx_diagram_request_links_diagram ON diagram_request_links (diagram_id);
