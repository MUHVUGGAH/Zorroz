# from camoufox.sync_api import Camoufox

# import asyncio
# from camoufox.async_api import AsyncCamoufox

# URLS = [
#     "https://httpbin.org/headers",
#     "https://httpbin.org/user-agent",
#     "https://httpbin.org/ip",
# ]


# async def scrape(page, url: str) -> dict:
#     await page.goto(url)
#     body = await page.inner_text("body")
#     return {"url": url, "body": body[:300]}


# async def main():
#     async with AsyncCamoufox(headless=True) as browser:
#         context = await browser.new_context()

#         pages = [await context.new_page() for _ in URLS]
#         results = await asyncio.gather(*[scrape(p, u) for p, u in zip(pages, URLS)])

#         for r in results:
#             print(f"\n--- {r['url']} ---")
#             print(r["body"])


# if __name__ == "__main__":
#     asyncio.run(main())

# 1. Enable 'humanize' to activate the natural motion algorithm.
# # You can pass True for default settings or a float for max duration in seconds.
# with Camoufox(humanize=2.0) as browser:
#     page = browser.new_page()
#     page.goto("https://camoufox.com/tests/buttonclick")

#     # 2. Perform a standard move or click. 
#     # Camoufox will automatically 'humanize' the path taken to these coordinates.
#     # It avoids straight lines and uses distance-aware trajectories.
#     # page.mouse.move(500, 500)
#     # page.mouse.click(500, 500)

#     # 3. You can also click directly on an element.
#     # The movement to the element's center will be humanized.
#     page.click("button#submit-id")

#     page.wait_for_timeout(3000)

# 'humanize' ensures the path to the dynamic button is natural
# with Camoufox(humanize=True) as browser:
#     page = browser.new_page()
#     page.goto("https://camoufox.com/tests/buttonclick")

#     # Define the locator once. It will "re-find" the button every time you use it.
#     button = page.get_by_role("button", name="Click me!")
#     # Alternatively: button = page.locator("button.button")

#     for i in range(10):
#         # 1. Playwright automatically waits for the button to be 'actionable'
#         # 2. Camoufox intercepts the click and moves the mouse in a human curve
#         button.click()
#         print(f"Clicked {i+1} times")
        
#         # Small random delay to look more human between clicks
#         page.wait_for_timeout(500) 