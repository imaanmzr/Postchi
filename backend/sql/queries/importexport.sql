-- name: InsertImportedCollection :one
INSERT INTO collections (workspace_id, parent_id, name, description, sort_order, variables, headers, auth, pre_request_script, test_script, created_by)
VALUES (@workspace_id, @parent_id, @name, @description, @sort_order, @variables, @headers, @auth, @pre_request_script, @test_script, @created_by)
RETURNING id;
