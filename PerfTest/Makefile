BUILD_DIR :=./build
EC2_KEY :=~/CSCipv6.pem
EC2_IP :=172.31.120.9
EC2_PATH :=/home/ubuntu/TestApps
EC2_ADDR :=ubuntu@ec2-13-221-33-53.compute-1.amazonaws.com
#CSC-ipv6-ubuntu2
EC2_IP2 :=3.89.69.226
EC2_ADDR2 :=ubuntu@ec2-3-89-69-226.compute-1.amazonaws.com
#CSC-ipv6-ubuntu3:	
EC2_IP3 :=54.198.168.204
EC2_ADDR3 :=ubuntu@ec2-54-198-168-204.compute-1.amazonaws.com

.PHONY: all clean build deploy run stop

all: build

buildIpv4:
	@echo "Building lambda: $@"
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/perfTestIpv6_ipv4 ./*.go

buildDualStack:
	@echo "Building lambda: $@"
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/perfTestIpv6_dualstack ./*.go

clean:
	@echo "Cleaning build directory"
	rm -rf $(BUILD_DIR)

deployIpv4: buildIpv4
	# rsync -avz -e "ssh -i $(EC2_KEY)" ./build/perfTestIpv6_ipv4 ec2-user@$(EC2_IP2):$(EC2_PATH)
	# ssh -i $(EC2_KEY) ec2-user@$(EC2_IP2) "chmod 755 $(EC2_PATH)/perfTestIpv6_ipv4"
	scp -i $(EC2_KEY) ./build/perfTestIpv6_ipv4 $(EC2_ADDR2):$(EC2_PATH)

	# rsync -avz -e "ssh -i $(EC2_KEY)" ./build/perfTestIpv6_ipv4 ec2-user@$(EC2_IP3):$(EC2_PATH)
	# ssh -i $(EC2_KEY) ec2-user@$(EC2_IP3) "chmod 755 $(EC2_PATH)/perfTestIpv6_ipv4"
	scp -i $(EC2_KEY) ./build/perfTestIpv6_ipv4 $(EC2_ADDR3):$(EC2_PATH)

deployDualStack: buildDualStack
	# rsync -avz -e "ssh -i $(EC2_KEY)" ./build/perfTestIpv6_dualstack ec2-user@$(EC2_IP2):$(EC2_PATH)
	# ssh -i $(EC2_KEY) ec2-user@$(EC2_IP2) "chmod 755 $(EC2_PATH)/perfTestIpv6_dualstack"
	scp -i $(EC2_KEY) ./build/perfTestIpv6_dualstack $(EC2_ADDR2):$(EC2_PATH)

	# rsync -avz -e "ssh -i $(EC2_KEY)" ./build/perfTestIpv6_dualstack ec2-user@$(EC2_IP3):$(EC2_PATH)
	# ssh -i $(EC2_KEY) ec2-user@$(EC2_IP3) "chmod 755 $(EC2_PATH)/perfTestIpv6_dualstack"
	scp -i $(EC2_KEY) ./build/perfTestIpv6_dualstack $(EC2_ADDR3):$(EC2_PATH)

#./dax-metrics -test=CacheHitDefault -op=GetItem
run:
	ssh -i $(EC2_KEY) ec2-user@$(EC2_IP2) "ls -la $(EC2_PATH)"

stop:
	ssh -i $(EC2_KEY) ec2-user@$(EC2_IP2) "ls -la $(EC2_PATH)"