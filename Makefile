.PHONY: run terraforming-mars

terraforming-mars: 
	go build -o build ./...

run: terraforming-mars
	build/terraforming-mars