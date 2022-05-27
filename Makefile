run0:
	env $$(cat .env | xargs) go run main.go reposts -u https://vk.com/wall-149859311_975
