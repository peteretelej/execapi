# Exec API

Execute commands and run scripts via HTTP APIs.

Exec API enables the execution of predefined commands and scripts on servers via HTTP APIs. Intended for legitimate administrative and automation purposes, it simplifies remote server management, offering a streamlined way for developers, DevOps, and IT professionals to interact with systems through standardized web protocols. Please use with proper access controls to ensure secure operation, see [Best Practice](./#Best-Practice) section below for recommend usage.


## Running ExecAPI

- Copy `config.json.sample` to `config.json`
- Update the key value (used for authorizing API requests), and the app details as necessary
- Run execapi. Either by,
    - Grabbing the latest release from the [Releases page](https://github.com/peteretelej/execapi/releases)
    - Or, building and running `execapi`

Building and running
``` sh
go build -o execapi .

./execapi
```

Run Exec API on the server you'd like to execute commands or scripts.

## Usage

Assuming you have an application at `/home/user/apps/appx` that has a script `deploy.sh` in the directory, you can use the [./config.json.sample](./config.json.sample) as the config.json.

Update `apps` to add different options as desired.

### Requesting command execution

```sh
curl -X POST http://localhost:8080/run/appx -H "Authorization: Bearer EXECAPI_KEY"
```
This will attempt to run the script for the app named "appx".

Responds with http 200 (success) if command execution succeeds, or http 400 (with failure error) on failure.

Request full execution logs with `?verbose=1`, for example:

```sh
curl -X POST http://localhost:8080/run/appx?verbose=1 -H "Authorization: Bearer EXECAPI_KEY"
```

### Windows?
Yes, this also works on Windows, simply provide the directory path in the config.json eg `"dir": "C:\\myapps\\appx"`, and the executable as the script eg `"script": "commandx.exe"`.

## Best Practice
Please consider configuring Exec API to only be accessible by intended clients. While using pre-defined commands via `config.json` restricts what can be done, it still allows a client to initiate process execution (even though intended) on your servers.

- Access control: For example, using tailscale and ufw to only allow desired access:
  - [Secure your Ubuntu server with Tailscale](https://tailscale.com/kb/1077/secure-server-ubuntu-18-04/)
  - [Tailscale GitHub Action](https://github.com/tailscale/github-action)
- Logging and Monitoring: Set up logging and monitoring to keep track of access
- Use HTTPS: Secure communication between the client and server with HTTPS

# License
[MIT](./LICENSE)
