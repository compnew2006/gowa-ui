import base64
import re

data = "iVBORw0KGgoAAAANSUhEUgAAAgAAAAIAAQMAAADOtka5AAAABlBMVEX///8AAABVwtN+AAAGPUlEQVR42uydO9Kryg6FRRE49BAYCkODoTEUD8EhAdW61VpLTbf/W6dOcKpQoE7+bRu+HdAPPZaE5MiRI0eOHDn++/HSOi55f0T0Ell0L/jFPh31X289Nvtq3ct8ymK3fBMQCVD/MV31Ipn0rBfZl6uqnu/Puuv1+i6HFOPYDcvh9yUgDmBRna56y15m1Y/YtR8DyHJsMp8NLttkuK1SExAV4Ivb5oFNDlvqdY1P16vCJ1ATEBiAX8XWcb1WFduxreO57sNbkX+aBwl4DmC7sn3NhyxczjhcsVWrX+InbgJiAXwB26//7s//M7IS8Cig4+x4yBvsoM0/zbzWvuwmQAICAfBYL1vA1UayDfjelVVhMe1S7SeYwbghAbEAsmrvh4zbcT1OzVQqM/wXWryD45mAxwG8VquNxAlQt9zl8DsnPflHzBut06GawZKASIB6iwV3jq3auIrtWGRT3GL/R1vA9ochhgTEAdg6Vn/INg+kxQ9sAlwM49mObf4nYgsJCATAlgsbyQI/aoFXqRuwtqBCC8r6nXMCggHqJmvXVudS8eQ90o7DtS11uCqL7jrsygl4HlBHacZttWOr//KRjZyC+A9je/gf6+T4JiAWYK2P9a2H57ZKO3Hfbg6tO/yXXTlVhvhBAp4HvFV32kE+HS76L1jOliJhiG/3NT4nIBbArNoLnPl+1jCHGE3QA7esKpgcZdyUE/A4oN5ZEOOBq7kjYYLAaztxYQZv7pT+TKQEPA2w/IcnlbEd18N1RYgWx6nlnW2OcOH/bOsJeBwAV9OetcA5AU4Y6qmPnCoCCxExxPdKQDTAxsCBuoqAW7Xtyly5ePKMxm7DrpyA5wH2PG0BC3Mj9ZHL0vR2k7YJYNuxXVLmMwGRAC7YgXOCJ29U43jCxAwnE4q0ozYB4QAUszIaxPirtpULaaS0Xdku+SYgFMDMIRfTQUUnXYhdZlf40GLqJM0JiAKwwB3F5HBc1PdhipiRFEE0yEMMpddkJSAAAKIOmwc8YyfozbtQwVJdFfuNIVrpT+cERADAK2FuhELz5pU0hQgk6RunwyjmSUAAgKWx9NaUCwMHNuwTVDzqRjGkPQkIBmDgQC0N8oH6HEEhZiqBszVe5r+OZwKeB8DGxa7Msh2esRQx+7VUgSAoOw0eSwKeB1ChDBWILdkWFIJxy0yXpy/NKR1i6wmIAGA2BNoOSz+b4GNnLY8nTCxw4FVbg8QyAc8D+poAprF8H0b6mdVXfqqKL/wERAL0litdFeYtkfDSe1awapmykQREArwYvwNTCeXKXv04n28WgaBAUhkwGvPOCYgAQOAczgkVrmWQz3WuJoziOxmWgDCAF6o/bK3S46S0jscp85a9Fn0osEtABIC4UgBFWG+XPyLvhfQl9T52/lqIdv+tc03Aw4DX1zIeNh1g8aKWoKuss0sYhaf99FPnmoCnAdyVzXJFqMcFV/zENi0TCkToeCYgGODVlZSzltUrBGbvlkS9Vp0O4GhJQDTAXedavCOZBV4RTWhUSrMuD99+ExAJ0BlA0oIDpimAq8lIn+/KZjENoq4EBACwh04TXNHGvSUCXsOsPg9wpSQgFACbbMuGeA+r3yIsOjX流动"

# Keep only characters matching standard base64 (A-Z, a-z, 0-9, +, /, =)
clean_data = "".join(re.findall(r'[A-Za-z0-9+/=]', data))

missing_padding = len(clean_data) % 4
if missing_padding:
    clean_data += '=' * (4 - missing_padding)

try:
    decoded = base64.b64decode(clean_data)
    with open("scratch/scratch_qr.png", "wb") as f:
        f.write(decoded)
    print("Decoded successfully. Size:", len(decoded))
except Exception as e:
    print("Error:", e)
