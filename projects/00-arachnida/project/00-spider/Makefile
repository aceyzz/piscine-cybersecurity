NAME = spider
SRCS = $(wildcard *.go)

all: $(NAME)

$(NAME): $(SRCS) go.sum
	go build -o $(NAME) .

go.mod:
	go mod init $(NAME)

go.sum: | go.mod
	go mod tidy

clean:
	rm -f $(NAME)

fclean: clean
	rm -rf data

re: fclean all

test: all
	bash ../tests/spider.sh $(CURDIR)/$(NAME)

.PHONY: all clean fclean re test
