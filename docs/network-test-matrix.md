# Network Test Matrix

Run these tests from a machine **other than** the one hosting the Postchi backend, against a real external API (e.g. `https://jsonplaceholder.typicode.com`).

## Prerequisites

1. Set `CORS_ORIGINS` to include the browser origin (e.g. `http://192.168.1.10:3000`)
2. Set `NUXT_PUBLIC_API_URL` to the browser-reachable API URL, **or** use same-origin reverse proxy (leave empty)
3. Backend must be able to reach the public internet (Docker: `extra_hosts: host.docker.internal` for host targets)

See also [documentation-linking.md](documentation-linking.md) for git doc sync and catalog smoke tests in your environment.


| #   | Method | Target                                                      | Body / headers                         | Expected                    | Pass? | Notes |
| --- | ------ | ----------------------------------------------------------- | -------------------------------------- | --------------------------- | ----- | ----- |
| 1   | GET    | `https://jsonplaceholder.typicode.com/posts?userId=1`       | -                                      | 200, array of posts         |       |       |
| 2   | POST   | `https://jsonplaceholder.typicode.com/posts`                | JSON `{"title":"test","body":"x","userId":1}` | 201, body echoed + id |       |       |
| 3   | POST   | `https://jsonplaceholder.typicode.com/posts`                | multipart form-data + file field       | 201 (JSONPlaceholder accepts multipart) |       |       |
| 4   | PUT    | `https://jsonplaceholder.typicode.com/posts/1`              | JSON body                              | 200                         |       |       |
| 5   | DELETE | `https://jsonplaceholder.typicode.com/posts/1`              | -                                      | 200                         |       |       |
| 6   | GET    | `https://jsonplaceholder.typicode.com/posts/1`              | `Authorization: Bearer test-token-123` | 200 (auth ignored by API)   |       |       |




## WebSocket (collaboration)


| Check                                                                      | Expected               | Pass? |
| -------------------------------------------------------------------------- | ---------------------- | ----- |
| Connect to `/api/ws?workspace_id=...&access_token=...` from remote browser | Connection opens (101) |       |


## Documentation (optional)


| Check | Expected | Pass? |
| ----- | -------- | ----- |
| Open `/workspaces/:id/docs` from remote browser | Documentation workspace loads | |
| Open `/workspaces/:id/catalog` | API catalog lists collections and endpoints | |
| Request **Docs** tab shows linked pages after manual link | Preview modal and **Open doc** work | |




## Record results

- **Tester machine IP:**
- **Postchi UI URL:**
- **Postchi API URL:**
- **Date:**
- **Tester:**
