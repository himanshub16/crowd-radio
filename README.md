# Crowdsourced radio station
Share and listen to music recommended by community, together.

**What it contains?**
A basic boilerplate to create a web-service over it.
It follows a service/implementation/repository based architecture, which provides interface to higher layers and can have multiple implementations at lower layers.

The benefit of such arch is to provide flexibility to freely swap any module as required, as long as they conform with the interface.

An example of such architecture can be found [here](https://github.com/himanshub16/outbound-go).

Here,

| filename               | description                                                                                         |
|------------------------|-----------------------------------------------------------------------------------------------------|
| models.go              | data types corresponding to what to be saved in database                                            |
| repositories.go        | interface defining operations to be done on the database                                            |
| repositories_sqlite.go | sqlite implementation of the methods defined in repositories                                        |
| service.go             | Services/Methods provided by this application. Service methods are indirectly called by the client. |
| handlers.go            | HTTP handlers which handle requests and perform operations using service methods                    |
| main.go                | Execution starts here                                                                               |


Here is the flow:
- create sqlite repository in main.
- create a serviceImpl which takes the repository for each model and has a way to implement each of them.
- create an HTTP router and provide it the services.

- any request comes first to the router.
- router communicates with the service object.
- service object peforms query over the repository if required and return.
- router understands what the service layer told and returns that to the client.
- 

### How to use?
- clone this repo.
- `go build` (it will install required modules on it's own)

For `hot-reloading`,
```bash
go get -u github.com/codegangsta/gin
gin --port 5000 --appPort 3000
# go to localhost:5000/health?message=hello
```

No worries about `GOPATH`. We have gomodules to the rescue.
