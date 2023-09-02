import logging
import requests

from typing import List
from src import cloudflare
from src.utils import App
from src.colorlogs import ColoredLevelFormatter

logging.getLogger().setLevel(logging.INFO)
formatter = ColoredLevelFormatter("%(levelname)s: %(message)s")
console = logging.StreamHandler()
console.setFormatter(ColoredLevelFormatter("%(levelname)s: %(message)s"))
logger = logging.getLogger()
logger.addHandler(console)

def read_lists_ini():
    with open("lists.txt", "r") as file:
        adlist_urls = [url.strip() for url in file if url.strip()]
    
    return adlist_urls

if __name__ == "__main__":
    adlist_urls = read_lists_ini()
    adlist_name = "DNS Block List"
    app = App(adlist_name, adlist_urls)
