default:
	./hack/build.sh
image:
	./hack/build-image.sh
clean:
	./hack/cleanup.sh
fmt:
	./hack/gofmt.sh
