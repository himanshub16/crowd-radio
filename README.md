# Crowdsourced radio station

### How this works?
Checkout the slides at the link below.

https://docs.google.com/presentation/d/e/2PACX-1vQqA42MCq5GiVDXQ_O2DU1hfi1XxY1MvQ_sFfEq17x6Ue-We8JyaG1KJEGQh-sSjELzZM34cyx3w1T0/pub?start=false&loop=false&delayms=3000


### Instructions to run
- Clone this repo and the [frontend client](https://github.com/littlewonder/upnext-frontend).

- Copy `.env.tempalte` to `.env`, and edit the required variables. Make sure the cluster variables have a reachable IP address.
- Start `postgres`
```
docker-compose up postgres
```

- Start the discovery service.
```
docker-compose up discovery
```

- Start as many services as you want (either in same shell, or different shell, as per your convenience).
```
# for same shell
docker-compose up backend_3030 backend_4040 backend_5050

# for different shells
docker-compose up backend_3030
docker-compose up backend_4040
docker-compose up backend_5050
```

- Start `nginx` for load-balancing the frontend client with backend.
Make sure `nginx-load-balancer.conf` has the right IP addresses.
```
docker-compose up nginx
```

- Visit to the directory where you cloned the frontend, and run the command below. You might need to update `google login client ID` in the [frontend code](https://github.com/thelittlewonder/upnext-frontend/blob/master/src/components/SignInContainer.vue#L52).
```
npm install
PORT=8080 API_PORT=8000 npm run dev
```


---

**TODO**: Add more text describing how it works.
