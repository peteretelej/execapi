# go-deploy

Go Deploy helps deploy apps on your Linux servers!

## Running Go-Deploy
```
export GODEPLOY_KEY={api_key that will be used for bearer auth}

go build -o go-deploy .

./go-deploy
```

Run Go-Deploy on the server you'd like to deploy apps on.


## Usage
Assuming you have an application at `/home/user/apps/appx` that has a deploy script `deploy.sh`, you can use the [./config.json.sample](./config.json.sample) as the config.json for go deploy.

Update `apps` to add different options as desired.

### Requesting a deployment
```
POST https://go-deploy.test/deploy/appx -H "Authorization: Bearer GODEPLOY_KEY"
```
This will attempt to run the script for the app named "appx".

Respond with http 200 (success) if deployment succeeds, or http 400 (with failure error) on failure.

Request full deployment logs with `?verbose=1`, for example:
```
POST https://go-deploy.test/deploy/appx?verbose=1 -H "Authorization: Bearer GODEPLOY_KEY"
```


## Best Practise
Consider making godeploy only accessible to your CI/CD pipelines.

For example, using tailscale and ufw to only allow desired access:
- https://tailscale.com/kb/1077/secure-server-ubuntu-18-04/
- https://github.com/tailscale/github-action


