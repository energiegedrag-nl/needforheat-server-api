# NeedForHeat Server API
![GitHub License](https://img.shields.io/github/license/energietransitie/needforheat-server-api)
![Project Status badge](https://img.shields.io/badge/status-in%20progress-brightgreen)

API for NeedForHeat data collection platform to enable research.

> **Own deployment.** This repository is maintained by [energiegedrag-nl](https://github.com/energiegedrag-nl) and deployed under `energiegedrag.nl`. To run your own instance, replace `energiegedrag-nl` by your GitHub organisation and `energiegedrag.nl` by your domain (hosts used: `api.<domain>`, `manuals.<domain>`), consistently in configuration and build settings.

## Table of contents
- [Deploying](#deploying)
- [Developing](#developing)
- [Usage](#usage)
- [License](#license)
- [Citation](#citation)
- [Credits](#credits)

## Deploying
For our process to deploy the API to our public server, or update it, see these links:
- Deploy: https://github.com/energiegedrag-nl/needforheat-server-configuration#api
- Update: https://github.com/energiegedrag-nl/needforheat-server-configuration#updating

### Prerequisites
The NeedForHeat API is available as a Docker image.
You will need to [install Docker](https://docs.docker.com/engine/install/) to run it.

### Images
See all [available images](https://github.com/energiegedrag-nl/needforheat-server-api/pkgs/container/needforheat-server-api):
- Use the `latest` tag to get the latest stable release built from a tagged GitHub release. 
- Use the `main` tag to get the latest development release, built directly from the `main` branch.

### Docker Compose ([more information](https://docs.docker.com/compose/features-uses/))
```yaml
services:
  web:
    container_name: needforheat-server-api-web
    build: .
    ports:
      - 8080:8080
    volumes:
      - ./data:/data
    environment:
      - NFH_DSN=root:needforheat@tcp(db:3306)/needforheat
      - NFH_BASE_URL=http://localhost:8080
      - NFH_DOWNLOAD_TIME=04h00m # 04:00 UTC
    depends_on:
      - db

  db:
    container_name: needforheat-server-api-db
    image: mariadb:latest
    ports:
      - 3306:3306
    volumes:
      - data:/var/lib/mysql
    environment:
      - MYSQL_DATABASE=needforheat
      - MYSQL_ROOT_PASSWORD=needforheat

volumes:
  data:
```

## Developing

### Requirements
- [Go (minimum 1.22)](https://go.dev/dl/)
- [Docker](https://www.docker.com/products/docker-desktop)

### Running
Make sure Docker is running on your local machine, then start the service from a command line terminal from the root of this repository:
```shell
docker compose up --build
```

This generates log messages from both the `web` and the `db` component.
Just keep this running in your terminal.

The API is now available on http://localhost:8080/.

Create a new admin account to use the admin endpoints:
```shell
docker exec -i <container-name> needforheat-server-api admin create -n <name>
```
> Substitute `<container-name>` with the name of the container. Running it locally should be `needforheat-server-api-web`

> Substitute `<name>` with the name of the admin account you want to create.

Example output of running the command, with `johndoe` as admin name:
```text
Admin "johndoe" created. Authorization token: eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJUd29tZXNBUEl2MiIsInN1YiI6IjQiLCJleHAiOjE3MTU3MzEyMDAsIm5iZiI6MTY4NDE1MjA4OSwiaWF0IjoxNjg0MTUyMDg5LCJraW5kIjoiYWRtaW5Ub2tlbiJ9.N_uhPhLsaTq0DVGVPhdfU6Hd2VD0Zb8QxesTaWeILlNkkjQ9Vuxpwe0sfi3Vj0GJgyin2ZilPE6AS-makGm2cg
```

Copy the authorization token without any spaces.

When finished, type the `Ctrl + C` key in the first terminal. The container state is 
preserved, and to restart, simply run `docker compose up --build` again.

To completely remove all docker containers created above:
```shell
docker compose rm
```

To delete the saved data, remove the `data` directory in the root of this repository.

### Folder structure

This repository tries to implement a DDD approach. While some elements are still too tightly coupled to really call it DDD, the structure still tries te represent DDD as best as possible.

| Folder       | Purpose                                                                   |
| ------------ | ------------------------------------------------------------------------- |
| .github      | GitHub Actions workflows and config files                                 |
| cmd          | CLI commands                                                              |
| docs         | Additional documentation                                                  |
| handlers     | HTTP handlers for API endpoints                                           |
| internal     | Utitilities that are not exposed outside of this package                  |
| repositories | Repositories for domain models. Contains al DB logic                      |
| services     | Services tie repositories and subservices together to perform operations. |
| swaggerdocs  | Swagger UI and OpenAPI spec.                                              |
| needforheat  | Domain models and logic.                                                  |

### Model diagram

To re-generate the model diagram:
```shell
docker run -i --rm vranac/erd < docs/model.er > docs/model.pdf
```

## Usage

### Tokens
Most end points require a session token to be provided in an authorization
bearer HTTP header. These end points are marked with a 'lock' symbol. Click
on the 'Authorize' button at the upper right of the page, or click on the 
'lock' symbol at the end point, and paste the session token in the value field.
Subsequent calls done through http://localhost:8080/docs will then use the
session token.

There are currently three types of tokens:
- Admin: Used by administrators to manage resources.
- Account: Used by an account to manage its resources.
- Device: Used by a measurement device to upload measurements.

### Managing admins and cloudfeeds
When the container is running, lookup it's name.

Run the following command to see info about how to manage admins:
```shell
docker exec <container-name> needforheat-server-api admin --help
```

Run the following command to see info about how to manage cloudfeeds:
```shell
docker exec <container-name> needforheat-server-api cloudfeed --help
```

### Administrators on our servers
Contact an administrator to get admin access to the API:
- Henri ter Hofte

## License
This software is available under the [Apache 2.0 license](./LICENSE), 
Copyright 2021 [Research group Energy Transition, Windesheim University of 
Applied Sciences](https://windesheim.nl/energietransitie) 

## Citation

If you use this repository in your research or work, please cite the following pre-print, which describes the overall NeedForHeat DataGear system of which this repository is a part:

> Ter Hofte, H., & van Ravenzwaaij, N. (2025). *NeedForHeat DataGear: An Open Monitoring System to Accelerate the Residential Heating Transition*. arXiv preprint arXiv:2509.06927. https://doi.org/10.48550/arXiv.2509.06927

**Note:** This is a pre-print submitted on 8 Sep 2025 and has not yet been peer-reviewed. For the full paper, see [https://arxiv.org/abs/2509.06927](https://arxiv.org/abs/2509.06927).

## Credits
Author:
* Henri ter Hofte · [@henriterhofte](https://github.com/henriterhofte)

This repository continues [energietransitie/needforheat-server-api](https://github.com/energietransitie/needforheat-server-api), made by the Research group Energy Transition at Windesheim University of Applied Sciences.
Thanks to the authors and contributors of the original:
* Nick van Ravenzwaaij · [@n-vr](https://github.com/n-vr)
* Arjan peddemors · [@arpe](https://github.com/arpe)
* Harris Mesic · [@labhatorian](https://github.com/Labhatorian)

We use and gratefully aknowlegde the efforts of the makers of the following source code and libraries:

- [chi](https://github.com/go-chi/chi), by Peter Kieltyka, Google Inc, licensed under [MIT license](https://github.com/go-chi/chi/blob/master/LICENSE)
- [gorm](https://gorm.io), by Jinzhu, licensed under [MIT license](https://github.com/go-gorm/gorm/blob/master/License)
- [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql), by go-sql-driver, licensed under [Mozilla Public License 2.0](https://github.com/go-sql-driver/mysql/blob/master/LICENSE)
- [logrus](https://github.com/sirupsen/logrus), by Simon Eskildsen, licensed under [MIT license](https://github.com/sirupsen/logrus/blob/master/LICENSE)
- [golang-jwt/jwt](https://github.com/golang-jwt/jwt), by Dave Grijalva, licensed under [MIT license](https://github.com/golang-jwt/jwt/blob/main/LICENSE)
- [crc16](https://github.com/sigurn/crc16), by sigurn, licensed under [MIT license](https://github.com/sigurn/crc16/blob/master/LICENSE)
- [swagger-ui](https://github.com/swagger-api/swagger-ui), by SmartBear Software Inc., licensed under [Apache 2.0 license](https://github.com/swagger-api/swagger-ui/blob/master/LICENSE)

This `README` uses [badges](https://github.com/badges/shields/blob/master/LICENSE), by [Shield.io](https://github.com/badges), licensed under [CC0 v1.0 Universal](https://github.com/badges/shields/blob/master/LICENSE)

### AI assistance

Since November 2026, parts of the maintenance of this repository have been done with [Claude Code](https://claude.com/claude-code) (Anthropic), under the author's direction and review. Commits made this way carry a `Co-Authored-By: Claude …` trailer. The author remains responsible for all content.
