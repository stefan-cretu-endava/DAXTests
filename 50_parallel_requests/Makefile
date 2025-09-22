BUILD_DIR := ./build
EC2_KEY := ~/CSCipv6.pem
EC2_IP := 54.198.168.204
EC2_PATH := /home/ubuntu/TestApps/
EC2_ADDR := ubuntu@ec2-100-26-207-158.compute-1.amazonaws.com
EXE_NAME := 50ParallelRequests

.PHONY: all clean build deploy run stop

all: build

build:
	@echo "Building lambda: $@"
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(EXE_NAME) ./*.go

clean:
	@echo "Cleaning build directory"
	rm -rf $(BUILD_DIR)

deploy: build
	# rsync -avz -e "ssh -i $(EC2_KEY)" ./build/* ec2-user@$(EC2_IP):$(EC2_PATH)
	# ssh -i $(EC2_KEY) ec2-user@$(EC2_IP) "chmod 755 $(EC2_PATH)/*"
	scp -i $(EC2_KEY) ./build/$(EXE_NAME) $(EC2_ADDR):~/TestApps/

#./dax-metrics -test=CacheHitDefault -op=GetItem
run:
	ssh -i $(EC2_KEY) ec2-user@$(EC2_IP) "ls -la $(EC2_PATH)"

stop:
	ssh -i $(EC2_KEY) ec2-user@$(EC2_IP) "ls -la $(EC2_PATH)"