from ctypes import *

class Timex(Structure):
    _fields_ = [("modes", c_int),       
                ("offset", c_long),     
                ("freq", c_long),       
                ("maxerror", c_long),
                ("esterror", c_long),
                ("status", c_int),
                ("constant", c_long),
                ("precision", c_long),
                ("tolerance", c_long),
                ("tv_sec", c_long),
                ("tv_usec", c_long),
                ("tick", c_long),
                ("ppsfreq", c_long),    
                ("jitter", c_long),
                ("shift", c_int),
                ("stabil", c_long),
                ("jitcnt", c_long),
                ("calcnt", c_long),
                ("errecnt", c_long),
                ("stbcnt", c_long),
                ("tai", c_int)]
    _comments_ = ("Mode selector", 
                  "Time offset, microseconds",
                  "Frequency offset in ppm where 1 lsb = 2^-16 ppm",
                  "Maximum error (microseconds)",
                  "Estimated error (microseconds)",
                  "Clock command/status",
                  "PLL (phase-locked loop) time constant",
                  "Clock precision (microseconds, read-only)",
                  "Clock frequency tolerance (read-only), 1 lsb = 2^-16 ppm",
                  "Time, seconds",
                  "Time, microseconds",
                  "Microseconds between clock ticks",
                  "PPS (pulse per second) frequency 1 lsb = 2^-16 ppm",
                  "PPS jitter (read-only)",
                  "PPS interval duration",
                  "PPS stability (read-only)",
                  "PPS count of jitter limit exceeded events (read-only)",
                  "PPS count of calibration intervals (read-only)",
                  "PPS count of calibration errors read-only)",
                  "PPS count of stability limit exceeded events (read-only)",
                  "TAI offset, as set by previous ADJ_TAI operation (seconds, read-only")
class Stat (Structure):
    _fields_ = [("STA_PLL", c_uint, 1),
                ("STA_PPSFREQ", c_uint, 1),
                ("STA_PPSTIME", c_uint, 1),
                ("STA_INS", c_uint, 1),
                ("STA_DEL", c_uint, 1),
                ("STA_UNSYNC", c_uint, 1),
                ("STA_FREQHOLD", c_uint, 1),
                ("STA_PPSSIGNAL", c_uint, 1),
                ("STA_PPSJITTER", c_uint, 1),
                ("STA_PPSWANDER", c_uint, 1),
                ("STA_PPSERROR", c_uint, 1),
                ("STA_PPSERROR", c_uint, 1),
                ("STA_NANO", c_uint, 1),
                ("STA_MODE", c_uint, 1),
                ("STA_CLK", c_uint, 1)]

class Status(Union):
    _fields_ = [("status", c_int),
                ("stat", Stat)]

BOLD = "\033[1m"
UNDER = "\033[4m"
END = "\033[0m"
rvs = ["TIME_OK", "TIME_INS", "TIME_DEL", "TIME_OOP", "TIME_WAIT", "TIME_ERROR"]
libc = cdll.LoadLibrary("libc.so.6")
t = Timex()
pt = pointer(t)
rv = libc.adjtimex(pt)
if rv >= 0:
    print(BOLD + f"adj_timex function call returned {rvs[rv]}" + END)
else:
    print(BOLD + f"adj_timex function call failed with {rv}" + END)

st = Status()
a = getattr(t, "status")
setattr(st, "status", getattr(t, "status"))

flags = []
for i in range(len(st.stat._fields_)):
    name = f"{st.stat._fields_[i][0]}"
    val = st.stat.__getattribute__(name)
    if val > 0:
        flags.append(name)

if len(flags) > 0:
    print(BOLD + f"Status flags: {flags}" + END)


print(UNDER + "Timex structure" + END)
for i in range(len(t._fields_)):
    print(f"{t._fields_[i][0]}:\t{t.__getattribute__(t._fields_[i][0])}\t{t._comments_[i]}".expandtabs(16))

