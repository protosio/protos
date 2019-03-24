# Protos - developing an application


## Create a local application ##

- create a file containing the  JSON payload that will be used to create the application (./payload.json)
```
{
	"installer-id": "apps.protos.io/gandi-dns",
	"installer-version": "0.0.1",
	"name": "gandi-dns",
	"installer-metadata": {
		"name": "gandi-dns",
		"provides": ["dns"],
		 "capabilities" : [
        	 	{
          			"Name" : "ResourceProvider"
        		},
    			{
          			"Name" : "GetInformation"
        		}
      		],
		"platformid": "gandi-dns:0.0.1"
	}
}
```

- create application via the dev API
```
curl -vX POST http://localhost:8080/api/v1/dev/apps -d @payload.json --header "Content-Type: application/json"
```

- using the app ID from the previous request, associate your development container to the newly created app
```
curl -vX POST http://localhost:8080/api/v1/dev/apps/<app id>/container -d '{"id": "<container id>"}' --header "Content-Type: application/json"
```