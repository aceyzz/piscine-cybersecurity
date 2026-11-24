SPIDER       = spider
SCORPION     = scorpion
SPIDER_DIR   = 00-spider
SCORPION_DIR = 01-scorpion

all: $(SPIDER) $(SCORPION)

$(SPIDER):
	$(MAKE) -C $(SPIDER_DIR)
	cp $(SPIDER_DIR)/$(SPIDER) $@

$(SCORPION):
	$(MAKE) -C $(SCORPION_DIR)
	cp $(SCORPION_DIR)/$(SCORPION) $@

clean:
	$(MAKE) -C $(SPIDER_DIR) clean
	$(MAKE) -C $(SCORPION_DIR) clean

fclean:
	$(MAKE) -C $(SPIDER_DIR) fclean
	$(MAKE) -C $(SCORPION_DIR) fclean
	rm -f $(SPIDER) $(SCORPION)
	rm -rf data

re: fclean all

test: all
	@status=0; \
	bash tests/spider.sh $(CURDIR)/$(SPIDER) || status=1; \
	bash tests/scorpion.sh $(CURDIR)/$(SCORPION) || status=1; \
	exit $$status

.PHONY: all clean fclean re test $(SPIDER) $(SCORPION)
