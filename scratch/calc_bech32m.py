import sys

CHARSET = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"

def bech32_polymod(values):
    generator = [0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3]
    chk = 1
    for value in values:
        top = chk >> 25
        chk = ((chk & 0x1ffffff) << 5) ^ value
        for i in range(5):
            chk ^= generator[i] if ((top >> i) & 1) else 0
    return chk

def bech32_hrp_expand(hrp):
    return [ord(x) >> 5 for x in hrp] + [0] + [ord(x) & 31 for x in hrp]

def create_checksum(hrp, data, spec):
    values = bech32_hrp_expand(hrp) + data
    const = 0x2bc830a3 if spec == "bech32m" else 1
    polymod = bech32_polymod(values + [0, 0, 0, 0, 0, 0]) ^ const
    return [(polymod >> 5 * (5 - i)) & 31 for i in range(6)]

def convertbits(data, frombits, tobits, pad=True):
    acc = 0
    bits = 0
    ret = []
    maxv = (1 << tobits) - 1
    max_acc = (1 << (frombits + tobits - 1)) - 1
    for value in data:
        if value < 0 or value >> frombits:
            return None
        acc = ((acc << frombits) | value) & max_acc
        bits += frombits
        while bits >= tobits:
            bits -= tobits
            ret.append((acc >> bits) & maxv)
    if pad:
        if bits:
            ret.append((acc << (tobits - bits)) & maxv)
    elif bits >= frombits or ((acc << (tobits - bits)) & maxv):
        return None
    return ret

# EVM address: 0x9d7a8cc0976455d73e22e44df8676cc45ac055ba
hex_addr = "9d7a8cc0976455d73e22e44df8676cc45ac055ba"
data_bytes = list(bytes.fromhex(hex_addr))
data_5bit = convertbits(data_bytes, 8, 5)

for spec in ["bech32", "bech32m"]:
    for hrp in ["cosmos", "sovereign"]:
        checksum = create_checksum(hrp, data_5bit, spec)
        addr = hrp + "1" + "".join([CHARSET[d] for d in data_5bit + checksum])
        print(f"{spec.upper()} ({hrp}): {addr}")
