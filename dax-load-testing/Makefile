BUILD_DIR := ./build
EC2_KEY := ~/CSCipv6.pem
EC2_IP :=172.31.120.9
EC2_IP1 :=54.198.168.204
EC2_IP2 :=3.89.69.226
EC2_IP3 :=13.223.74.232
EC2_IP4 :=54.242.239.232
EC2_PATH :=/home/ubuntu/TestApps/
EC2_ADDR :=ubuntu@ec2-13-221-33-53.compute-1.amazonaws.com
EC2_ADDR1 :=ubuntu@ec2-54-198-168-204.compute-1.amazonaws.com
EC2_ADDR2 :=ubuntu@ec2-3-89-69-226.compute-1.amazonaws.com
EC2_ADDR3 :=ubuntu@ec2-13-223-74-232.compute-1.amazonaws.com
EC2_ADDR4 :=ubuntu@ec2-54-242-239-232.compute-1.amazonaws.com

.PHONY: all clean build deploy run stop

all: build

build:
	@echo "Building lambda: $@"
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/loadTestIPv6_dualstack_highCPU ./*.go

buildDualStack:
	@echo "Building lambda: $@"
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/loadTestIPv6_dualstack_highCPU ./*.go

buildipv4:
	@echo "Building lambda: $@"
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/loadTestIPv6_ipv4 ./*.go

buildDualStackTLS:
	@echo "Building lambda: $@"
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/loadTestIPv6_dualstack_highCPU_tls ./*.go

buildipv4TLS:
	@echo "Building lambda: $@"
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/loadTestIPv6_ipv4_tls ./*.go

clean:
	@echo "Cleaning build directory"
	rm -rf $(BUILD_DIR)

deploy: build
	# rsync -avz -e "ssh -i $(EC2_KEY)" ./build/* ec2-user@$(EC2_IP):$(EC2_PATH)
	# ssh -i $(EC2_KEY) ec2-user@$(EC2_IP) "chmod 755 $(EC2_PATH)/*"
	scp -i $(EC2_KEY) ./build/* $(EC2_ADDR):$(EC2_PATH)

deployDualStack: buildDualStack
	# rsync -avz -e "ssh -i $(EC2_KEY)" ./build/* ec2-user@$(EC2_IP1):$(EC2_PATH)
	# ssh -i $(EC2_KEY) ec2-user@$(EC2_IP1) "chmod 755 $(EC2_PATH)/*"
	scp -i $(EC2_KEY) ./build/loadTestIPv6_dualstack_highCPU $(EC2_ADDR1):$(EC2_PATH)
	# rsync -avz -e "ssh -i $(EC2_KEY)" ./build/* ec2-user@$(EC2_IP2):$(EC2_PATH)
	# ssh -i $(EC2_KEY) ec2-user@$(EC2_IP2) "chmod 755 $(EC2_PATH)/*"
	scp -i $(EC2_KEY) ./build/loadTestIPv6_dualstack_highCPU $(EC2_ADDR2):$(EC2_PATH)
	# rsync -avz -e "ssh -i $(EC2_KEY)" ./build/* ec2-user@$(EC2_IP3):$(EC2_PATH)
	# ssh -i $(EC2_KEY) ec2-user@$(EC2_IP3) "chmod 755 $(EC2_PATH)/*"
	scp -i $(EC2_KEY) ./build/loadTestIPv6_dualstack_highCPU $(EC2_ADDR3):$(EC2_PATH)
	# rsync -avz -e "ssh -i $(EC2_KEY)" ./build/* ec2-user@$(EC2_IP4):$(EC2_PATH)
	# ssh -i $(EC2_KEY) ec2-user@$(EC2_IP4) "chmod 755 $(EC2_PATH)/*"
	scp -i $(EC2_KEY) ./build/loadTestIPv6_dualstack_highCPU $(EC2_ADDR4):$(EC2_PATH)


deployDualStackTLS: buildDualStackTLS
	# rsync -avz -e "ssh -i $(EC2_KEY)" ./build/* ec2-user@$(EC2_IP1):$(EC2_PATH)
	# ssh -i $(EC2_KEY) ec2-user@$(EC2_IP1) "chmod 755 $(EC2_PATH)/*"
	scp -i $(EC2_KEY) ./build/loadTestIPv6_dualstack_highCPU_tls $(EC2_ADDR1):$(EC2_PATH)
	# rsync -avz -e "ssh -i $(EC2_KEY)" ./build/* ec2-user@$(EC2_IP2):$(EC2_PATH)
	# ssh -i $(EC2_KEY) ec2-user@$(EC2_IP2) "chmod 755 $(EC2_PATH)/*"
	scp -i $(EC2_KEY) ./build/loadTestIPv6_dualstack_highCPU_tls $(EC2_ADDR2):$(EC2_PATH)
	# rsync -avz -e "ssh -i $(EC2_KEY)" ./build/* ec2-user@$(EC2_IP3):$(EC2_PATH)
	# ssh -i $(EC2_KEY) ec2-user@$(EC2_IP3) "chmod 755 $(EC2_PATH)/*"
	scp -i $(EC2_KEY) ./build/loadTestIPv6_dualstack_highCPU_tls $(EC2_ADDR3):$(EC2_PATH)
	# rsync -avz -e "ssh -i $(EC2_KEY)" ./build/* ec2-user@$(EC2_IP4):$(EC2_PATH)
	# ssh -i $(EC2_KEY) ec2-user@$(EC2_IP4) "chmod 755 $(EC2_PATH)/*"
	scp -i $(EC2_KEY) ./build/loadTestIPv6_dualstack_highCPU_tls $(EC2_ADDR4):$(EC2_PATH)
	
deployIpv4: buildipv4
	# rsync -avz -e "ssh -i $(EC2_KEY)" ./build/* ec2-user@$(EC2_IP1):$(EC2_PATH)
	# ssh -i $(EC2_KEY) ec2-user@$(EC2_IP1) "chmod 755 $(EC2_PATH)/*"
	scp -i $(EC2_KEY) ./build/loadTestIPv6_ipv4 $(EC2_ADDR1):$(EC2_PATH)
	# rsync -avz -e "ssh -i $(EC2_KEY)" ./build/* ec2-user@$(EC2_IP2):$(EC2_PATH)
	# ssh -i $(EC2_KEY) ec2-user@$(EC2_IP2) "chmod 755 $(EC2_PATH)/*"
	scp -i $(EC2_KEY) ./build/loadTestIPv6_ipv4 $(EC2_ADDR2):$(EC2_PATH)
	# rsync -avz -e "ssh -i $(EC2_KEY)" ./build/* ec2-user@$(EC2_IP3):$(EC2_PATH)
	# ssh -i $(EC2_KEY) ec2-user@$(EC2_IP3) "chmod 755 $(EC2_PATH)/*"
	scp -i $(EC2_KEY) ./build/loadTestIPv6_ipv4 $(EC2_ADDR3):$(EC2_PATH)
	# rsync -avz -e "ssh -i $(EC2_KEY)" ./build/* ec2-user@$(EC2_IP4):$(EC2_PATH)
	# ssh -i $(EC2_KEY) ec2-user@$(EC2_IP4) "chmod 755 $(EC2_PATH)/*"
	scp -i $(EC2_KEY) ./build/loadTestIPv6_ipv4 $(EC2_ADDR4):$(EC2_PATH)

deployIpv4TLS: buildipv4TLS
	# rsync -avz -e "ssh -i $(EC2_KEY)" ./build/* ec2-user@$(EC2_IP1):$(EC2_PATH)
	# ssh -i $(EC2_KEY) ec2-user@$(EC2_IP1) "chmod 755 $(EC2_PATH)/*"
	scp -i $(EC2_KEY) ./build/loadTestIPv6_ipv4_tls $(EC2_ADDR1):$(EC2_PATH)
	# rsync -avz -e "ssh -i $(EC2_KEY)" ./build/* ec2-user@$(EC2_IP2):$(EC2_PATH)
	# ssh -i $(EC2_KEY) ec2-user@$(EC2_IP2) "chmod 755 $(EC2_PATH)/*"
	scp -i $(EC2_KEY) ./build/loadTestIPv6_ipv4_tls $(EC2_ADDR2):$(EC2_PATH)
	# rsync -avz -e "ssh -i $(EC2_KEY)" ./build/* ec2-user@$(EC2_IP3):$(EC2_PATH)
	# ssh -i $(EC2_KEY) ec2-user@$(EC2_IP3) "chmod 755 $(EC2_PATH)/*"
	scp -i $(EC2_KEY) ./build/loadTestIPv6_ipv4_tls $(EC2_ADDR3):$(EC2_PATH)
	# rsync -avz -e "ssh -i $(EC2_KEY)" ./build/* ec2-user@$(EC2_IP4):$(EC2_PATH)
	# ssh -i $(EC2_KEY) ec2-user@$(EC2_IP4) "chmod 755 $(EC2_PATH)/*"
	scp -i $(EC2_KEY) ./build/loadTestIPv6_ipv4_tls $(EC2_ADDR4):$(EC2_PATH)


#./dax-metrics -test=CacheHitDefault -op=GetItem
run:
	ssh -i $(EC2_KEY) ec2-user@$(EC2_IP) "ls -la $(EC2_PATH)"

stop:
	ssh -i $(EC2_KEY) ec2-user@$(EC2_IP) "ls -la $(EC2_PATH)"