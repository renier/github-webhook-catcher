# Github Webhook Catcher

A server that listens for webhook notifications and passes the body of the request off to a configured command. Intended to be used to listen to Github webhook notifications, but they could come from any kind of place. What drove this was the need to execute concourse.ci jobs when a webhook notification came in, [a feature that is not yet intrisically available in concourse](https://github.com/concourse/concourse/issues/331).

## Usage

```
Usage of github-webhook-catcher:
  -access-token string
    	If provided, any webhook notification must pass the same token in the query string of the request
  -command string
    	Command to execute once a webhook notification is received. Required
  -port string
    	Port to listen for webhook notifications on (default "8088")
  -source-host string
    	Source host where the webhook notification will come from (default "github.com")
  -tls-cert string
    	Path to TLS certificate used to support SSL encryption
  -tls-key string
    	Path to TLS key used to support SSL encryption
```

## Install

```
go get github.com/renier/github-webhook-catcher
```

There is a _systemd_ service file that can be used to install the catcher as a service. Simply copy the service file to `/lib/systemd/system/github-webhook-catcher.service`, then enable it:

```
systemctl enable github-webhook-catcher.service
```

Take a look at the service file for particulars like configuration parameters and expected location of the binary.