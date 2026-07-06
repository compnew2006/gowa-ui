import struct

try:
    with open("scratch_qr.png", "rb") as f:
        data = f.read()
    
    # Verify PNG signature
    if data[:8] != b'\x89PNG\r\n\x1a\n':
        print("Not a valid PNG file")
    else:
        # IHDR chunk starts at byte 12
        # Width (4 bytes) and Height (4 bytes) start at byte 16
        w, h = struct.unpack(">II", data[16:24])
        print(f"PNG dimensions: {w}x{h}")
except Exception as e:
    print("Error:", e)
