# Gun Deals API
## What is this?
[Gun.deals](https://gun.deals) doesn't offer any kind of public API for retrieving deals on products, so this project scrapes their pages and rehosts the data in an easily consumable format. 
---
## Features
- **Current Deals**: Gets the latest deals from the homepage, deals posted today, or deals from any category on gun.deals
- **Search**: Returns a list of possible matches along with their lowest price
- **Individual product**: Returns a list of the current best deals for any specified product
- **Coupons**: Gets a list of coupons, rebates, and promotions currently going on
---
## Installation
```bash
# Clone the repository
git clone https://github.com/That-Thing/GunDealsAPI.git

# Navigate to project directory
cd GunDealsAPI

# Install dependencies
go mod download

# Build the project
go build

# Run the server
./gundealsAPI
```
## Usage
### Commandline Arguments
- `-host`: Specifies the host address to bind the server to (default: "localhost")
- `-port`: Specifies the port number to listen on (default: 8080)

### API Endpoints
```
GET /relevant - Returns current deals from the homepage
GET /today - Returns deals from today
GET /category/{category} - Returns deals for a specific category
GET /search?q={query} - Search for products
GET /product/{id} - Get deals for a specific product
```
---
## Disclaimer
This project is not affiliated with Gun.deals. This API is intended for personal and non-commercial use only. Excessive requests may result in your IP being blocked by Gun.deals. Please use responsibly and consider the website's terms of service.