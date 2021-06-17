
EXECUTORS = \
	default \
	keptn \

all:
	for executor in $(EXECUTORS); \
	do \
		$(MAKE) -C $$executor; \
	done

# Run tests
test:
	for executor in $(EXECUTORS); \
	do \
		$(MAKE) -C $$executor test; \
	done

# Build manager binary
build: 
	for executor in $(EXECUTORS); \
	do \
		$(MAKE) -C $$executor build; \
	done

# Run go fmt against code
fmt:
	for executor in $(EXECUTORS); \
	do \
		$(MAKE) -C $$executor fmt; \
	done

# Run go vet against code
vet:
	for executor in $(EXECUTORS); \
	do \
		$(MAKE) -C $$executor vet; \
	done

# Build the docker image
docker-build: 
	for executor in $(EXECUTORS); \
	do \
		$(MAKE) -C $$executor docker-build; \
	done

# Push the docker image
docker-push:
	for executor in $(EXECUTORS); \
	do \
		$(MAKE) -C $$executor docker-push; \
	done
