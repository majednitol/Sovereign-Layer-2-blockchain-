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

# Decoded data 5-bit array from 'cosmos1n4agesyhv32aw03zu3xlsemvc3dvq4d635c5mq'
# cosmos1 n4agesyhv32aw03zu3xlsemvc3dvq4d6 35c5mq
data_str = "n4agesyhv32aw03zu3xlsemvc3dvq4d6"
data_5bit = [CHARSET.index(c) for c in data_str]

# Let's search prefixes of length 1 to 15, and check if any prefix + data matches 'uccuw0' (in standard bech32 or bech32m)
target_suffix = "uccuw0"
target_checksum = [CHARSET.index(c) for c in target_suffix]

# We will search by generating all possible 6-character HRPs, or checking a list of possible HRPs, or analyzing the polymod equation directly!
# Since we know the polymod equation is linear, we can easily find the prefix if we know it!
# Wait! Let's check common prefixes first:
common_prefixes = [
    "cosmos", "sovereign", "sov", "chain", "node", "addr", "account", "val", "validator",
    "wasm", "contract", "cw", "erc", "evm", "eth", "bridge", "relayer", "faucet", "user",
    "cosmosvaloper", "cosmosvalcons", "sovereignvaloper", "sovereignvalcons"
]

found = False
for spec in ["bech32", "bech32m"]:
    for hrp in common_prefixes:
        checksum = create_checksum(hrp, data_5bit, spec)
        if checksum == target_checksum:
            print(f"FOUND MATCH! HRP: {hrp}, Spec: {spec}")
            found = True

if not found:
    print("No match found in common prefixes. Let's do a broader search on prefix characters.")
    # We can solve the HRP expanding logic.
    # The polymod function is: bech32_polymod(bech32_hrp_expand(hrp) + data + checksum) ^ const == 0 (for verify)
    # Let's find HRP that satisfies this!
    # HRP expand of HRP (h_0, h_1, ..., h_{k-1}) is:
    # [h_0>>5, h_1>>5, ..., h_{k-1}>>5, 0, h_0&31, h_1&31, ..., h_{k-1}&31]
    # Since HRP characters are lowercase ascii, h_i>>5 is always 3 (since ord('a')=97=01100001_2, ord('z')=122=01111010_2, so 97>>5 = 3, 122>>5 = 3).
    # So h_i>>5 is always 3!
    # And h_i&31 is in [1, 26] (since ord('a')&31 = 1, ord('z')&31 = 26).
    # So we can represent HRP as h_i = 3 * 32 + x_i, where x_i is in [1, 26].
    # Let's search HRP of length 1 to 10 by brute-force since it's very fast in python!
    for length in range(1, 10):
        # We can optimize: we know the HRP characters are ascii lowercase.
        # Let's brute-force length 3 to 6:
        if length == 3:
            import itertools
            for p in itertools.product(range(97, 123), repeat=3):
                hrp = "".join(chr(c) for c in p)
                checksum = create_checksum(hrp, data_5bit, "bech32")
                if checksum == target_checksum:
                    print(f"FOUND MATCH (len 3): {hrp} (bech32)")
                checksum_m = create_checksum(hrp, data_5bit, "bech32m")
                if checksum_m == target_checksum:
                    print(f"FOUND MATCH (len 3): {hrp} (bech32m)")
        elif length == 4:
            import itertools
            for p in itertools.product(range(97, 123), repeat=4):
                hrp = "".join(chr(c) for c in p)
                checksum = create_checksum(hrp, data_5bit, "bech32")
                if checksum == target_checksum:
                    print(f"FOUND MATCH (len 4): {hrp} (bech32)")
                checksum_m = create_checksum(hrp, data_5bit, "bech32m")
                if checksum_m == target_checksum:
                    print(f"FOUND MATCH (len 4): {hrp} (bech32m)")
        elif length == 5:
            # We can run it quickly
            pass
