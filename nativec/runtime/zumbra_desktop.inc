#if defined(ZUMBRA_ENABLE_DESKTOP)

/* Z13 cross-platform desktop runtime. SDL3 is loaded dynamically so generated
 * executables remain standalone and can still run in deterministic headless mode. */
#define ZUMBRA_DESKTOP_SDL3 1
#include <ctype.h>
#include <fcntl.h>
#include <sys/types.h>
#if defined(_WIN32)
#include <direct.h>
#include <io.h>
#include <process.h>
#include <commdlg.h>
#include <shellapi.h>
#include <shlobj.h>
#ifndef PATH_MAX
#define PATH_MAX MAX_PATH
#endif
#define z_desktop_getcwd _getcwd
#define z_desktop_chdir _chdir
#define z_desktop_popen _popen
#define z_desktop_pclose _pclose
#define z_desktop_access _access
#else
#include <dirent.h>
#include <dlfcn.h>
#include <signal.h>
#include <sys/wait.h>
#include <unistd.h>
#ifndef PATH_MAX
#define PATH_MAX 4096
#endif
#define z_desktop_getcwd getcwd
#define z_desktop_chdir chdir
#define z_desktop_popen popen
#define z_desktop_pclose pclose
#define z_desktop_access access
#endif

static void *z_desktop_library_open(const char *name) {
#if defined(_WIN32)
    return (void *)LoadLibraryA(name);
#else
    return dlopen(name, RTLD_NOW | RTLD_LOCAL);
#endif
}
static void *z_desktop_library_symbol(void *library, const char *name) {
#if defined(_WIN32)
    return library == NULL ? NULL : (void *)GetProcAddress((HMODULE)library, name);
#else
    return library == NULL ? NULL : dlsym(library, name);
#endif
}
static void z_desktop_library_close(void *library) {
    if (library == NULL) return;
#if defined(_WIN32)
    FreeLibrary((HMODULE)library);
#else
    dlclose(library);
#endif
}

typedef struct ZDesktopAppNative ZDesktopAppNative;
typedef struct ZDesktopWindowNative ZDesktopWindowNative;
typedef struct ZDesktopTrayNative ZDesktopTrayNative;
typedef struct ZDesktopProcessNative ZDesktopProcessNative;
typedef struct ZDesktopEventNative ZDesktopEventNative;
struct ZUIContextNative;
#if defined(ZUMBRA_ENABLE_UI)
static void z_ui_dispatch_from_desktop(ZDesktopAppNative *app, ZDesktopEventNative *event);
static void z_ui_render_all(ZDesktopAppNative *app);
static bool z_ui_any_dirty(ZDesktopAppNative *app);
static void z_ui_release_renderer_resources(void *renderer);
#else
static void z_ui_dispatch_from_desktop(ZDesktopAppNative *app, ZDesktopEventNative *event) { (void)app; (void)event; }
static void z_ui_render_all(ZDesktopAppNative *app) { (void)app; }
static bool z_ui_any_dirty(ZDesktopAppNative *app) { (void)app; return false; }
static void z_ui_release_renderer_resources(void *renderer) { (void)renderer; }
#endif

typedef struct ZDesktopHandlerNative {
    char *name;
    ZValue handler;
    struct ZDesktopHandlerNative *next;
} ZDesktopHandlerNative;

typedef struct ZDesktopEventNative {
    char *type;
    int64_t window_id;
    int64_t timestamp;
    ZValue data;
    struct ZDesktopEventNative *next;
} ZDesktopEventNative;

typedef struct {
    ZValue receiver;
    const char *builtin;
} ZDesktopMethodNative;

struct ZDesktopWindowNative {
    ZDesktopAppNative *app;
    void *handle;
    void *renderer;
    void *frame_texture;
    int64_t frame_width, frame_height;
    bool vsync;
    int64_t id;
    char *title;
    int64_t width, height, pixel_width, pixel_height, x, y;
    double scale, density;
    bool open, visible, fullscreen;
    struct ZDesktopWindowNative *next;
};

struct ZDesktopTrayNative {
    ZDesktopAppNative *app;
    void *handle;
    void *menu;
    bool open;
    struct ZDesktopTrayNative *next;
};

struct ZDesktopProcessNative {
#if defined(_WIN32)
    HANDLE handle;
    DWORD pid;
#else
    pid_t pid;
#endif
    bool running;
    bool waited;
    int64_t exit_code;
};

struct ZDesktopAppNative {
    pthread_mutex_t mutex;
    bool headless, running, closed;
    bool run_thread_set;
    pthread_t run_thread;
    bool close_on_request, quit_on_last_window;
    int64_t poll_interval_ms;
    int64_t next_window_id;
    char *clipboard;
    ZDesktopWindowNative *windows;
    ZDesktopTrayNative *trays;
    ZDesktopHandlerNative *handlers;
    ZDesktopHandlerNative *shortcuts;
    ZDesktopEventNative *event_head, *event_tail;
    struct ZUIContextNative *ui_contexts;
    void *audio_stream;
    void *gamepads[2];
    uint32_t gamepad_ids[2];
};

/* -------- dictionary and value helpers -------- */
static ZValue z_desktop_dict_get(ZValue value, const char *key) {
    if (value.tag != ZV_DICT || value.as.dict == NULL) return z_null();
    for (size_t i = 0; i < value.as.dict->len; i++) {
        ZValue k = value.as.dict->keys[i];
        if (k.tag == ZV_STRING && strcmp(k.as.s, key) == 0) return value.as.dict->values[i];
    }
    return z_null();
}
static const char *z_desktop_opt_string(ZValue value, const char *key, const char *fallback) {
    ZValue found = z_desktop_dict_get(value, key);
    return found.tag == ZV_STRING ? found.as.s : fallback;
}
static int64_t z_desktop_opt_int(ZValue value, const char *key, int64_t fallback) {
    ZValue found = z_desktop_dict_get(value, key);
    if (found.tag == ZV_INT) return found.as.i;
    if (found.tag == ZV_UINT) return (int64_t)found.as.u;
    return fallback;
}
static bool z_desktop_opt_bool(ZValue value, const char *key, bool fallback) {
    ZValue found = z_desktop_dict_get(value, key);
    return found.tag == ZV_BOOL ? found.as.b : fallback;
}
static double z_desktop_opt_float(ZValue value, const char *key, double fallback) {
    ZValue found = z_desktop_dict_get(value, key);
    if (found.tag == ZV_FLOAT) return found.as.f;
    if (found.tag == ZV_INT) return (double)found.as.i;
    if (found.tag == ZV_UINT) return (double)found.as.u;
    return fallback;
}
static ZValue z_desktop_dict1(const char *a, ZValue av) {
    ZValue pair = z_pair(z_string(a), av); return z_dict_from(&pair, 1);
}
static ZValue z_desktop_dict2(const char *a, ZValue av, const char *b, ZValue bv) {
    ZValue pairs[2] = {z_pair(z_string(a), av), z_pair(z_string(b), bv)}; return z_dict_from(pairs, 2);
}
static ZValue z_desktop_dict3(const char *a, ZValue av, const char *b, ZValue bv, const char *c, ZValue cv) {
    ZValue pairs[3] = {z_pair(z_string(a), av), z_pair(z_string(b), bv), z_pair(z_string(c), cv)}; return z_dict_from(pairs, 3);
}
static int64_t z_desktop_now_ns(void) {
#if defined(_WIN32)
    FILETIME value; ULARGE_INTEGER ticks; GetSystemTimeAsFileTime(&value);
    ticks.LowPart=value.dwLowDateTime; ticks.HighPart=value.dwHighDateTime;
    return (int64_t)((ticks.QuadPart-INT64_C(116444736000000000))*100);
#else
    struct timespec ts; clock_gettime(CLOCK_REALTIME, &ts);
    return (int64_t)ts.tv_sec * INT64_C(1000000000) + (int64_t)ts.tv_nsec;
#endif
}
static void z_desktop_sleep_ms(int64_t timeout_ms) {
    if(timeout_ms<=0)return;
#if defined(_WIN32)
    Sleep((DWORD)timeout_ms);
#else
    struct timespec request={.tv_sec=(time_t)(timeout_ms/1000),.tv_nsec=(long)((timeout_ms%1000)*1000000)};
    nanosleep(&request,NULL);
#endif
}
static ZValue z_desktop_object(ZTag tag, ZKind kind, void *pointer) {
    ZValue value = z_value(tag, kind); value.as.p = pointer; return value;
}
static ZDesktopAppNative *z_desktop_expect_app(ZValue value) {
    if (value.tag != ZV_DESKTOP_APP || value.as.p == NULL) z_fatal("expected DesktopApp");
    return (ZDesktopAppNative *)value.as.p;
}
static ZDesktopWindowNative *z_desktop_expect_window(ZValue value) {
    if (value.tag != ZV_DESKTOP_WINDOW || value.as.p == NULL) z_fatal("expected DesktopWindow");
    return (ZDesktopWindowNative *)value.as.p;
}
static ZDesktopTrayNative *z_desktop_expect_tray(ZValue value) {
    if (value.tag != ZV_DESKTOP_TRAY || value.as.p == NULL) z_fatal("expected DesktopTray");
    return (ZDesktopTrayNative *)value.as.p;
}
static ZDesktopProcessNative *z_desktop_expect_process(ZValue value) {
    if (value.tag != ZV_DESKTOP_PROCESS || value.as.p == NULL) z_fatal("expected DesktopProcess");
    return (ZDesktopProcessNative *)value.as.p;
}

/* -------- SDL3 dynamic adapter -------- */
#if defined(ZUMBRA_DESKTOP_SDL3)
typedef struct SDL_Window SDL_Window;
typedef struct SDL_Surface SDL_Surface;
typedef struct SDL_IOStream SDL_IOStream;
typedef struct SDL_Tray SDL_Tray;
typedef struct SDL_TrayMenu SDL_TrayMenu;
typedef struct SDL_TrayEntry SDL_TrayEntry;
typedef struct SDL_Renderer SDL_Renderer;
typedef struct SDL_Texture SDL_Texture;
typedef struct SDL_Cursor SDL_Cursor;
typedef struct SDL_AudioStream SDL_AudioStream;
typedef struct SDL_Gamepad SDL_Gamepad;
typedef uint32_t SDL_AudioDeviceID;
typedef uint32_t SDL_JoystickID;
typedef struct { uint32_t format; int channels; int freq; } ZDesktopSDLAudioSpec;
typedef struct { float x,y,w,h; } ZDesktopSDLFRect;
typedef struct { int32_t x,y,w,h; } ZDesktopSDLRect;
typedef union ZDesktopSDLEvent { uint32_t type; uint8_t padding[128]; } ZDesktopSDLEvent;
typedef struct { uint32_t type, reserved; uint64_t timestamp; } ZDesktopSDLCommonEvent;
typedef struct { uint32_t type,reserved; uint64_t timestamp; uint32_t windowID; int32_t data1,data2; } ZDesktopSDLWindowEvent;
typedef struct { uint32_t type,reserved; uint64_t timestamp; uint32_t windowID,which,scancode,key; uint16_t mod,raw; bool down,repeat; } ZDesktopSDLKeyboardEvent;
typedef struct { uint32_t type,reserved; uint64_t timestamp; uint32_t windowID; const char *text; } ZDesktopSDLTextEvent;
typedef struct { uint32_t type,reserved; uint64_t timestamp; uint32_t windowID,which,state; float x,y,xrel,yrel; } ZDesktopSDLMouseMotionEvent;
typedef struct { uint32_t type,reserved; uint64_t timestamp; uint32_t windowID,which; uint8_t button; bool down; uint8_t clicks,padding; float x,y; } ZDesktopSDLMouseButtonEvent;
typedef struct { uint32_t type,reserved; uint64_t timestamp; uint32_t windowID,which; float x,y; uint32_t direction; float mouse_x,mouse_y; int32_t integer_x,integer_y; } ZDesktopSDLMouseWheelEvent;
typedef struct { uint32_t type,reserved; uint64_t timestamp; uint32_t windowID; float x,y; const char *source; const char *data; } ZDesktopSDLDropEvent;

typedef bool (*ZD_SDL_Init)(uint32_t); typedef void (*ZD_SDL_Quit)(void); typedef const char *(*ZD_SDL_GetError)(void);
typedef bool (*ZD_SDL_SetAppMetadata)(const char*,const char*,const char*);
typedef SDL_Window *(*ZD_SDL_CreateWindow)(const char*,int,int,uint64_t); typedef void (*ZD_SDL_DestroyWindow)(SDL_Window*); typedef uint32_t (*ZD_SDL_GetWindowID)(SDL_Window*);
typedef bool (*ZD_SDL_ShowWindow)(SDL_Window*); typedef bool (*ZD_SDL_HideWindow)(SDL_Window*); typedef const char *(*ZD_SDL_GetWindowTitle)(SDL_Window*); typedef bool (*ZD_SDL_SetWindowTitle)(SDL_Window*,const char*);
typedef bool (*ZD_SDL_GetWindowSize)(SDL_Window*,int*,int*); typedef bool (*ZD_SDL_GetWindowSizeInPixels)(SDL_Window*,int*,int*); typedef bool (*ZD_SDL_SetWindowSize)(SDL_Window*,int,int);
typedef bool (*ZD_SDL_GetWindowPosition)(SDL_Window*,int*,int*); typedef bool (*ZD_SDL_SetWindowPosition)(SDL_Window*,int,int); typedef bool (*ZD_SDL_SetWindowFullscreen)(SDL_Window*,bool); typedef uint64_t (*ZD_SDL_GetWindowFlags)(SDL_Window*);
typedef bool (*ZD_SDL_MaximizeWindow)(SDL_Window*); typedef bool (*ZD_SDL_MinimizeWindow)(SDL_Window*); typedef bool (*ZD_SDL_RestoreWindow)(SDL_Window*); typedef bool (*ZD_SDL_RaiseWindow)(SDL_Window*);
typedef float (*ZD_SDL_GetWindowDisplayScale)(SDL_Window*); typedef float (*ZD_SDL_GetWindowPixelDensity)(SDL_Window*);
typedef bool (*ZD_SDL_PollEvent)(ZDesktopSDLEvent*); typedef bool (*ZD_SDL_WaitEvent)(ZDesktopSDLEvent*); typedef bool (*ZD_SDL_WaitEventTimeout)(ZDesktopSDLEvent*,int32_t); typedef bool (*ZD_SDL_PushEvent)(ZDesktopSDLEvent*); typedef const char *(*ZD_SDL_GetKeyName)(uint32_t);
typedef bool (*ZD_SDL_StartTextInput)(SDL_Window*); typedef bool (*ZD_SDL_StopTextInput)(SDL_Window*); typedef SDL_Cursor *(*ZD_SDL_CreateSystemCursor)(int); typedef bool (*ZD_SDL_SetCursor)(SDL_Cursor*); typedef void (*ZD_SDL_DestroyCursor)(SDL_Cursor*);
typedef bool (*ZD_SDL_SetClipboardText)(const char*); typedef char *(*ZD_SDL_GetClipboardText)(void); typedef void (*ZD_SDL_free)(void*);
typedef SDL_IOStream *(*ZD_SDL_IOFromFile)(const char*,const char*); typedef SDL_Surface *(*ZD_SDL_LoadBMP_IO)(SDL_IOStream*,bool); typedef void (*ZD_SDL_DestroySurface)(SDL_Surface*); typedef bool (*ZD_SDL_SetWindowIcon)(SDL_Window*,SDL_Surface*);
typedef SDL_Tray *(*ZD_SDL_CreateTray)(SDL_Surface*,const char*); typedef SDL_TrayMenu *(*ZD_SDL_CreateTrayMenu)(SDL_Tray*); typedef SDL_TrayEntry *(*ZD_SDL_InsertTrayEntryAt)(SDL_TrayMenu*,int,const char*,uint32_t); typedef void (*ZD_SDL_SetTrayEntryCallback)(SDL_TrayEntry*,void(*)(void*,SDL_TrayEntry*),void*); typedef void (*ZD_SDL_SetTrayTooltip)(SDL_Tray*,const char*); typedef void (*ZD_SDL_DestroyTray)(SDL_Tray*);
typedef SDL_Renderer *(*ZD_SDL_CreateRenderer)(SDL_Window*,const char*); typedef void (*ZD_SDL_DestroyRenderer)(SDL_Renderer*);
typedef bool (*ZD_SDL_SetRenderDrawColor)(SDL_Renderer*,uint8_t,uint8_t,uint8_t,uint8_t); typedef bool (*ZD_SDL_SetRenderDrawBlendMode)(SDL_Renderer*,uint32_t); typedef bool (*ZD_SDL_SetRenderClipRect)(SDL_Renderer*,const ZDesktopSDLRect*); typedef bool (*ZD_SDL_RenderClear)(SDL_Renderer*);
typedef bool (*ZD_SDL_RenderFillRect)(SDL_Renderer*,const ZDesktopSDLFRect*); typedef bool (*ZD_SDL_RenderRect)(SDL_Renderer*,const ZDesktopSDLFRect*);
typedef bool (*ZD_SDL_RenderLine)(SDL_Renderer*,float,float,float,float); typedef bool (*ZD_SDL_RenderPresent)(SDL_Renderer*); typedef bool (*ZD_SDL_RenderDebugText)(SDL_Renderer*,float,float,const char*);
typedef SDL_Texture *(*ZD_SDL_CreateTextureFromSurface)(SDL_Renderer*,SDL_Surface*); typedef SDL_Surface *(*ZD_SDL_CreateSurfaceFrom)(int,int,uint32_t,void*,int); typedef SDL_Surface *(*ZD_SDL_RenderReadPixels)(SDL_Renderer*,const ZDesktopSDLRect*); typedef SDL_Surface *(*ZD_SDL_ConvertSurface)(SDL_Surface*,uint32_t); typedef bool (*ZD_SDL_RenderTexture)(SDL_Renderer*,SDL_Texture*,const ZDesktopSDLFRect*,const ZDesktopSDLFRect*); typedef void (*ZD_SDL_DestroyTexture)(SDL_Texture*); typedef bool (*ZD_SDL_GetTextureSize)(SDL_Texture*,float*,float*);
typedef SDL_Texture *(*ZD_SDL_CreateTexture)(SDL_Renderer*,uint32_t,int,int,int); typedef bool (*ZD_SDL_UpdateTexture)(SDL_Texture*,const ZDesktopSDLRect*,const void*,int); typedef bool (*ZD_SDL_SetRenderLogicalPresentation)(SDL_Renderer*,int,int,int); typedef bool (*ZD_SDL_SetRenderVSync)(SDL_Renderer*,int);
typedef const bool *(*ZD_SDL_GetKeyboardState)(int*); typedef SDL_JoystickID *(*ZD_SDL_GetGamepads)(int*); typedef SDL_Gamepad *(*ZD_SDL_OpenGamepad)(SDL_JoystickID); typedef void (*ZD_SDL_CloseGamepad)(SDL_Gamepad*); typedef bool (*ZD_SDL_GetGamepadButton)(SDL_Gamepad*,int); typedef bool (*ZD_SDL_GamepadConnected)(SDL_Gamepad*);
typedef SDL_AudioStream *(*ZD_SDL_OpenAudioDeviceStream)(SDL_AudioDeviceID,const ZDesktopSDLAudioSpec*,void*,void*); typedef bool (*ZD_SDL_PutAudioStreamData)(SDL_AudioStream*,const void*,int); typedef int (*ZD_SDL_GetAudioStreamQueued)(SDL_AudioStream*); typedef bool (*ZD_SDL_ResumeAudioStreamDevice)(SDL_AudioStream*); typedef void (*ZD_SDL_DestroyAudioStream)(SDL_AudioStream*);

typedef struct {
    void *lib; bool initialized; char error[512];
    ZD_SDL_Init Init; ZD_SDL_Quit Quit; ZD_SDL_GetError GetError; ZD_SDL_SetAppMetadata SetAppMetadata;
    ZD_SDL_CreateWindow CreateWindow; ZD_SDL_DestroyWindow DestroyWindow; ZD_SDL_GetWindowID GetWindowID; ZD_SDL_ShowWindow ShowWindow; ZD_SDL_HideWindow HideWindow; ZD_SDL_GetWindowTitle GetWindowTitle; ZD_SDL_SetWindowTitle SetWindowTitle;
    ZD_SDL_GetWindowSize GetWindowSize; ZD_SDL_GetWindowSizeInPixels GetWindowSizeInPixels; ZD_SDL_SetWindowSize SetWindowSize; ZD_SDL_GetWindowPosition GetWindowPosition; ZD_SDL_SetWindowPosition SetWindowPosition; ZD_SDL_SetWindowFullscreen SetWindowFullscreen; ZD_SDL_GetWindowFlags GetWindowFlags;
    ZD_SDL_MaximizeWindow MaximizeWindow; ZD_SDL_MinimizeWindow MinimizeWindow; ZD_SDL_RestoreWindow RestoreWindow; ZD_SDL_RaiseWindow RaiseWindow; ZD_SDL_GetWindowDisplayScale GetWindowDisplayScale; ZD_SDL_GetWindowPixelDensity GetWindowPixelDensity;
    ZD_SDL_PollEvent PollEvent; ZD_SDL_WaitEvent WaitEvent; ZD_SDL_WaitEventTimeout WaitEventTimeout; ZD_SDL_PushEvent PushEvent; ZD_SDL_GetKeyName GetKeyName; ZD_SDL_StartTextInput StartTextInput; ZD_SDL_StopTextInput StopTextInput; ZD_SDL_CreateSystemCursor CreateSystemCursor; ZD_SDL_SetCursor SetCursor; ZD_SDL_DestroyCursor DestroyCursor; ZD_SDL_SetClipboardText SetClipboardText; ZD_SDL_GetClipboardText GetClipboardText; ZD_SDL_free Free;
    ZD_SDL_IOFromFile IOFromFile; ZD_SDL_LoadBMP_IO LoadBMP_IO; ZD_SDL_DestroySurface DestroySurface; ZD_SDL_SetWindowIcon SetWindowIcon;
    ZD_SDL_CreateTray CreateTray; ZD_SDL_CreateTrayMenu CreateTrayMenu; ZD_SDL_InsertTrayEntryAt InsertTrayEntryAt; ZD_SDL_SetTrayEntryCallback SetTrayEntryCallback; ZD_SDL_SetTrayTooltip SetTrayTooltip; ZD_SDL_DestroyTray DestroyTray;
    ZD_SDL_CreateRenderer CreateRenderer; ZD_SDL_DestroyRenderer DestroyRenderer; ZD_SDL_SetRenderDrawColor SetRenderDrawColor; ZD_SDL_SetRenderDrawBlendMode SetRenderDrawBlendMode; ZD_SDL_SetRenderClipRect SetRenderClipRect; ZD_SDL_RenderClear RenderClear; ZD_SDL_RenderFillRect RenderFillRect; ZD_SDL_RenderRect RenderRect; ZD_SDL_RenderLine RenderLine; ZD_SDL_RenderPresent RenderPresent; ZD_SDL_RenderDebugText RenderDebugText; ZD_SDL_CreateTextureFromSurface CreateTextureFromSurface; ZD_SDL_CreateSurfaceFrom CreateSurfaceFrom; ZD_SDL_RenderReadPixels RenderReadPixels; ZD_SDL_ConvertSurface ConvertSurface; ZD_SDL_RenderTexture RenderTexture; ZD_SDL_DestroyTexture DestroyTexture; ZD_SDL_GetTextureSize GetTextureSize;
    ZD_SDL_CreateTexture CreateTexture; ZD_SDL_UpdateTexture UpdateTexture; ZD_SDL_SetRenderLogicalPresentation SetRenderLogicalPresentation; ZD_SDL_SetRenderVSync SetRenderVSync; ZD_SDL_GetKeyboardState GetKeyboardState; ZD_SDL_GetGamepads GetGamepads; ZD_SDL_OpenGamepad OpenGamepad; ZD_SDL_CloseGamepad CloseGamepad; ZD_SDL_GetGamepadButton GetGamepadButton; ZD_SDL_GamepadConnected GamepadConnected; ZD_SDL_OpenAudioDeviceStream OpenAudioDeviceStream; ZD_SDL_PutAudioStreamData PutAudioStreamData; ZD_SDL_GetAudioStreamQueued GetAudioStreamQueued; ZD_SDL_ResumeAudioStreamDevice ResumeAudioStreamDevice; ZD_SDL_DestroyAudioStream DestroyAudioStream;
} ZDesktopSDL;
static ZDesktopSDL z_desktop_sdl;
#define ZD_LOAD_REQUIRED(field, symbol, type) do { z_desktop_sdl.field=(type)z_desktop_library_symbol(z_desktop_sdl.lib,symbol); if(z_desktop_sdl.field==NULL){snprintf(z_desktop_sdl.error,sizeof(z_desktop_sdl.error),"missing SDL3 symbol %s",symbol);return false;} } while(0)
#define ZD_LOAD_OPTIONAL(field, symbol, type) do { z_desktop_sdl.field=(type)z_desktop_library_symbol(z_desktop_sdl.lib,symbol); } while(0)
static bool z_desktop_sdl_load(void) {
    if (z_desktop_sdl.initialized) return true;
    memset(&z_desktop_sdl, 0, sizeof(z_desktop_sdl));
#if defined(_WIN32)
    const char *names[] = {"SDL3.dll", "libSDL3.dll", NULL};
#elif defined(__APPLE__)
    const char *names[] = {"@executable_path/../Frameworks/libSDL3.0.dylib", "@executable_path/../Frameworks/libSDL3.dylib", "libSDL3.0.dylib", "libSDL3.dylib", "/Library/Frameworks/SDL3.framework/SDL3", NULL};
#else
    const char *names[] = {"libSDL3.so.0", "libSDL3.so", NULL};
#endif
    for (size_t i=0; names[i] != NULL; i++) { z_desktop_sdl.lib=z_desktop_library_open(names[i]); if(z_desktop_sdl.lib!=NULL)break; }
    if (z_desktop_sdl.lib == NULL) { snprintf(z_desktop_sdl.error,sizeof(z_desktop_sdl.error),"SDL3 shared library was not found"); return false; }
    ZD_LOAD_REQUIRED(Init,"SDL_Init",ZD_SDL_Init); ZD_LOAD_REQUIRED(Quit,"SDL_Quit",ZD_SDL_Quit); ZD_LOAD_REQUIRED(GetError,"SDL_GetError",ZD_SDL_GetError);
    ZD_LOAD_OPTIONAL(SetAppMetadata,"SDL_SetAppMetadata",ZD_SDL_SetAppMetadata);
    ZD_LOAD_REQUIRED(CreateWindow,"SDL_CreateWindow",ZD_SDL_CreateWindow); ZD_LOAD_REQUIRED(DestroyWindow,"SDL_DestroyWindow",ZD_SDL_DestroyWindow); ZD_LOAD_REQUIRED(GetWindowID,"SDL_GetWindowID",ZD_SDL_GetWindowID);
    ZD_LOAD_REQUIRED(ShowWindow,"SDL_ShowWindow",ZD_SDL_ShowWindow); ZD_LOAD_REQUIRED(HideWindow,"SDL_HideWindow",ZD_SDL_HideWindow); ZD_LOAD_REQUIRED(GetWindowTitle,"SDL_GetWindowTitle",ZD_SDL_GetWindowTitle); ZD_LOAD_REQUIRED(SetWindowTitle,"SDL_SetWindowTitle",ZD_SDL_SetWindowTitle);
    ZD_LOAD_REQUIRED(GetWindowSize,"SDL_GetWindowSize",ZD_SDL_GetWindowSize); ZD_LOAD_REQUIRED(GetWindowSizeInPixels,"SDL_GetWindowSizeInPixels",ZD_SDL_GetWindowSizeInPixels); ZD_LOAD_REQUIRED(SetWindowSize,"SDL_SetWindowSize",ZD_SDL_SetWindowSize); ZD_LOAD_REQUIRED(GetWindowPosition,"SDL_GetWindowPosition",ZD_SDL_GetWindowPosition); ZD_LOAD_REQUIRED(SetWindowPosition,"SDL_SetWindowPosition",ZD_SDL_SetWindowPosition); ZD_LOAD_REQUIRED(SetWindowFullscreen,"SDL_SetWindowFullscreen",ZD_SDL_SetWindowFullscreen); ZD_LOAD_REQUIRED(GetWindowFlags,"SDL_GetWindowFlags",ZD_SDL_GetWindowFlags);
    ZD_LOAD_REQUIRED(MaximizeWindow,"SDL_MaximizeWindow",ZD_SDL_MaximizeWindow); ZD_LOAD_REQUIRED(MinimizeWindow,"SDL_MinimizeWindow",ZD_SDL_MinimizeWindow); ZD_LOAD_REQUIRED(RestoreWindow,"SDL_RestoreWindow",ZD_SDL_RestoreWindow); ZD_LOAD_REQUIRED(RaiseWindow,"SDL_RaiseWindow",ZD_SDL_RaiseWindow); ZD_LOAD_REQUIRED(GetWindowDisplayScale,"SDL_GetWindowDisplayScale",ZD_SDL_GetWindowDisplayScale); ZD_LOAD_REQUIRED(GetWindowPixelDensity,"SDL_GetWindowPixelDensity",ZD_SDL_GetWindowPixelDensity);
    ZD_LOAD_REQUIRED(PollEvent,"SDL_PollEvent",ZD_SDL_PollEvent); ZD_LOAD_REQUIRED(WaitEvent,"SDL_WaitEvent",ZD_SDL_WaitEvent); ZD_LOAD_REQUIRED(WaitEventTimeout,"SDL_WaitEventTimeout",ZD_SDL_WaitEventTimeout); ZD_LOAD_REQUIRED(PushEvent,"SDL_PushEvent",ZD_SDL_PushEvent); ZD_LOAD_REQUIRED(GetKeyName,"SDL_GetKeyName",ZD_SDL_GetKeyName); ZD_LOAD_REQUIRED(StartTextInput,"SDL_StartTextInput",ZD_SDL_StartTextInput); ZD_LOAD_REQUIRED(StopTextInput,"SDL_StopTextInput",ZD_SDL_StopTextInput); ZD_LOAD_OPTIONAL(CreateSystemCursor,"SDL_CreateSystemCursor",ZD_SDL_CreateSystemCursor); ZD_LOAD_OPTIONAL(SetCursor,"SDL_SetCursor",ZD_SDL_SetCursor); ZD_LOAD_OPTIONAL(DestroyCursor,"SDL_DestroyCursor",ZD_SDL_DestroyCursor); ZD_LOAD_REQUIRED(SetClipboardText,"SDL_SetClipboardText",ZD_SDL_SetClipboardText); ZD_LOAD_REQUIRED(GetClipboardText,"SDL_GetClipboardText",ZD_SDL_GetClipboardText); ZD_LOAD_REQUIRED(Free,"SDL_free",ZD_SDL_free);
    ZD_LOAD_OPTIONAL(IOFromFile,"SDL_IOFromFile",ZD_SDL_IOFromFile); ZD_LOAD_OPTIONAL(LoadBMP_IO,"SDL_LoadBMP_IO",ZD_SDL_LoadBMP_IO); ZD_LOAD_OPTIONAL(DestroySurface,"SDL_DestroySurface",ZD_SDL_DestroySurface); ZD_LOAD_OPTIONAL(SetWindowIcon,"SDL_SetWindowIcon",ZD_SDL_SetWindowIcon);
    ZD_LOAD_OPTIONAL(CreateTray,"SDL_CreateTray",ZD_SDL_CreateTray); ZD_LOAD_OPTIONAL(CreateTrayMenu,"SDL_CreateTrayMenu",ZD_SDL_CreateTrayMenu); ZD_LOAD_OPTIONAL(InsertTrayEntryAt,"SDL_InsertTrayEntryAt",ZD_SDL_InsertTrayEntryAt); ZD_LOAD_OPTIONAL(SetTrayEntryCallback,"SDL_SetTrayEntryCallback",ZD_SDL_SetTrayEntryCallback); ZD_LOAD_OPTIONAL(SetTrayTooltip,"SDL_SetTrayTooltip",ZD_SDL_SetTrayTooltip); ZD_LOAD_OPTIONAL(DestroyTray,"SDL_DestroyTray",ZD_SDL_DestroyTray);
    ZD_LOAD_REQUIRED(CreateRenderer,"SDL_CreateRenderer",ZD_SDL_CreateRenderer); ZD_LOAD_REQUIRED(DestroyRenderer,"SDL_DestroyRenderer",ZD_SDL_DestroyRenderer); ZD_LOAD_REQUIRED(SetRenderDrawColor,"SDL_SetRenderDrawColor",ZD_SDL_SetRenderDrawColor); ZD_LOAD_OPTIONAL(SetRenderDrawBlendMode,"SDL_SetRenderDrawBlendMode",ZD_SDL_SetRenderDrawBlendMode); ZD_LOAD_OPTIONAL(SetRenderClipRect,"SDL_SetRenderClipRect",ZD_SDL_SetRenderClipRect); ZD_LOAD_REQUIRED(RenderClear,"SDL_RenderClear",ZD_SDL_RenderClear); ZD_LOAD_REQUIRED(RenderFillRect,"SDL_RenderFillRect",ZD_SDL_RenderFillRect); ZD_LOAD_REQUIRED(RenderRect,"SDL_RenderRect",ZD_SDL_RenderRect); ZD_LOAD_REQUIRED(RenderLine,"SDL_RenderLine",ZD_SDL_RenderLine); ZD_LOAD_REQUIRED(RenderPresent,"SDL_RenderPresent",ZD_SDL_RenderPresent); ZD_LOAD_OPTIONAL(RenderDebugText,"SDL_RenderDebugText",ZD_SDL_RenderDebugText); ZD_LOAD_OPTIONAL(CreateTextureFromSurface,"SDL_CreateTextureFromSurface",ZD_SDL_CreateTextureFromSurface); ZD_LOAD_OPTIONAL(CreateSurfaceFrom,"SDL_CreateSurfaceFrom",ZD_SDL_CreateSurfaceFrom); ZD_LOAD_OPTIONAL(RenderReadPixels,"SDL_RenderReadPixels",ZD_SDL_RenderReadPixels); ZD_LOAD_OPTIONAL(ConvertSurface,"SDL_ConvertSurface",ZD_SDL_ConvertSurface); ZD_LOAD_OPTIONAL(RenderTexture,"SDL_RenderTexture",ZD_SDL_RenderTexture); ZD_LOAD_OPTIONAL(DestroyTexture,"SDL_DestroyTexture",ZD_SDL_DestroyTexture); ZD_LOAD_OPTIONAL(GetTextureSize,"SDL_GetTextureSize",ZD_SDL_GetTextureSize);
    ZD_LOAD_OPTIONAL(CreateTexture,"SDL_CreateTexture",ZD_SDL_CreateTexture); ZD_LOAD_OPTIONAL(UpdateTexture,"SDL_UpdateTexture",ZD_SDL_UpdateTexture); ZD_LOAD_OPTIONAL(SetRenderLogicalPresentation,"SDL_SetRenderLogicalPresentation",ZD_SDL_SetRenderLogicalPresentation); ZD_LOAD_OPTIONAL(SetRenderVSync,"SDL_SetRenderVSync",ZD_SDL_SetRenderVSync); ZD_LOAD_OPTIONAL(GetKeyboardState,"SDL_GetKeyboardState",ZD_SDL_GetKeyboardState); ZD_LOAD_OPTIONAL(GetGamepads,"SDL_GetGamepads",ZD_SDL_GetGamepads); ZD_LOAD_OPTIONAL(OpenGamepad,"SDL_OpenGamepad",ZD_SDL_OpenGamepad); ZD_LOAD_OPTIONAL(CloseGamepad,"SDL_CloseGamepad",ZD_SDL_CloseGamepad); ZD_LOAD_OPTIONAL(GetGamepadButton,"SDL_GetGamepadButton",ZD_SDL_GetGamepadButton); ZD_LOAD_OPTIONAL(GamepadConnected,"SDL_GamepadConnected",ZD_SDL_GamepadConnected); ZD_LOAD_OPTIONAL(OpenAudioDeviceStream,"SDL_OpenAudioDeviceStream",ZD_SDL_OpenAudioDeviceStream); ZD_LOAD_OPTIONAL(PutAudioStreamData,"SDL_PutAudioStreamData",ZD_SDL_PutAudioStreamData); ZD_LOAD_OPTIONAL(GetAudioStreamQueued,"SDL_GetAudioStreamQueued",ZD_SDL_GetAudioStreamQueued); ZD_LOAD_OPTIONAL(ResumeAudioStreamDevice,"SDL_ResumeAudioStreamDevice",ZD_SDL_ResumeAudioStreamDevice); ZD_LOAD_OPTIONAL(DestroyAudioStream,"SDL_DestroyAudioStream",ZD_SDL_DestroyAudioStream);
    if (!z_desktop_sdl.Init(0x00000020u|0x00000010u|0x00002000u)) { snprintf(z_desktop_sdl.error,sizeof(z_desktop_sdl.error),"%s",z_desktop_sdl.GetError()); return false; }
    z_desktop_sdl.initialized=true; return true;
}

static void z_desktop_refresh_gamepads(ZDesktopAppNative *app) {
    if(app==NULL||app->headless||z_desktop_sdl.GetGamepads==NULL||z_desktop_sdl.OpenGamepad==NULL)return;
    for(int slot=0;slot<2;slot++){
        SDL_Gamepad*pad=(SDL_Gamepad*)app->gamepads[slot];
        if(pad!=NULL&&z_desktop_sdl.GamepadConnected!=NULL&&!z_desktop_sdl.GamepadConnected(pad)){
            if(z_desktop_sdl.CloseGamepad!=NULL)z_desktop_sdl.CloseGamepad(pad);
            app->gamepads[slot]=NULL;app->gamepad_ids[slot]=0;
        }
    }
    int count=0;SDL_JoystickID*ids=z_desktop_sdl.GetGamepads(&count);if(ids==NULL)return;
    for(int i=0;i<count;i++){
        bool open=false;for(int slot=0;slot<2;slot++)if(app->gamepad_ids[slot]==ids[i])open=true;
        if(open)continue;
        for(int slot=0;slot<2;slot++)if(app->gamepads[slot]==NULL){SDL_Gamepad*pad=z_desktop_sdl.OpenGamepad(ids[i]);if(pad!=NULL){app->gamepads[slot]=pad;app->gamepad_ids[slot]=ids[i];}break;}
    }
    if(z_desktop_sdl.Free!=NULL)z_desktop_sdl.Free(ids);else free(ids);
}

static SDL_AudioStream *z_desktop_audio_stream(ZDesktopAppNative *app) {
    if(app==NULL||app->headless)return NULL;
    if(app->audio_stream!=NULL)return (SDL_AudioStream*)app->audio_stream;
    if(z_desktop_sdl.OpenAudioDeviceStream==NULL)return NULL;
    ZDesktopSDLAudioSpec spec={UINT32_C(0x8010),1,44100};
    SDL_AudioStream*stream=z_desktop_sdl.OpenAudioDeviceStream(UINT32_MAX,&spec,NULL,NULL);
    if(stream!=NULL&&z_desktop_sdl.ResumeAudioStreamDevice!=NULL)(void)z_desktop_sdl.ResumeAudioStreamDevice(stream);
    app->audio_stream=stream;return stream;
}

static SDL_Cursor *z_desktop_system_cursors[3]={NULL,NULL,NULL};
static void z_desktop_destroy_system_cursors(void){
    if(z_desktop_sdl.DestroyCursor){for(int i=0;i<3;i++){if(z_desktop_system_cursors[i]){z_desktop_sdl.DestroyCursor(z_desktop_system_cursors[i]);z_desktop_system_cursors[i]=NULL;}}}
}
#if defined(ZUMBRA_ENABLE_UI)
static void z_desktop_set_system_cursor(int kind){
    if(!z_desktop_sdl.CreateSystemCursor||!z_desktop_sdl.SetCursor)return;
    if(kind<0||kind>2)kind=0;
    int system_kind=kind==1?11:(kind==2?1:0);
    if(!z_desktop_system_cursors[kind])z_desktop_system_cursors[kind]=z_desktop_sdl.CreateSystemCursor(system_kind);
    if(z_desktop_system_cursors[kind])z_desktop_sdl.SetCursor(z_desktop_system_cursors[kind]);
}
#endif

static SDL_Surface *z_desktop_sdl_load_bmp(const char *path) {
    if(path==NULL||*path=='\0'||z_desktop_sdl.IOFromFile==NULL||z_desktop_sdl.LoadBMP_IO==NULL)return NULL;
    SDL_IOStream *stream=z_desktop_sdl.IOFromFile(path,"rb"); if(stream==NULL)return NULL; return z_desktop_sdl.LoadBMP_IO(stream,true);
}
#else
static bool z_desktop_sdl_load(void) { return false; }
#endif

#define Z_DESKTOP_POLL_SENTINEL UINT32_C(0x7F00)
#define Z_DESKTOP_WAKE_EVENT UINT32_C(0x8000)

static void z_desktop_wake_app(ZDesktopAppNative *app) {
#if defined(ZUMBRA_DESKTOP_SDL3)
    if (app == NULL || app->headless || !app->running || app->closed || !z_desktop_sdl.initialized || z_desktop_sdl.PushEvent == NULL) return;
    if (app->run_thread_set && pthread_equal(pthread_self(), app->run_thread)) return;
    ZDesktopSDLEvent wake; memset(&wake, 0, sizeof(wake)); wake.type = Z_DESKTOP_WAKE_EVENT;
    (void)z_desktop_sdl.PushEvent(&wake);
#else
    (void)app;
#endif
}

/* -------- event queue and handlers -------- */
static void z_desktop_queue(ZDesktopAppNative *app, const char *type, int64_t window_id, ZValue data) {
    ZDesktopEventNative *event=(ZDesktopEventNative *)z_alloc(sizeof(*event));
    event->type=z_strdup(type); event->window_id=window_id; event->timestamp=z_desktop_now_ns(); event->data=data; event->next=NULL;
    pthread_mutex_lock(&app->mutex);
    bool was_empty=app->event_head==NULL;
    if(app->event_tail!=NULL)app->event_tail->next=event;else app->event_head=event;
    app->event_tail=event;
    pthread_mutex_unlock(&app->mutex);
    if(was_empty)z_desktop_wake_app(app);
}
static ZDesktopEventNative *z_desktop_dequeue(ZDesktopAppNative *app) {
    pthread_mutex_lock(&app->mutex); ZDesktopEventNative *event=app->event_head;
    if(event!=NULL){app->event_head=event->next;if(app->event_head==NULL)app->event_tail=NULL;}
    pthread_mutex_unlock(&app->mutex); return event;
}
static ZValue z_desktop_event_value(ZDesktopEventNative *event) {
    if(event==NULL)return z_null();
    ZValue pairs[4]={
        z_pair(z_string("type"),z_string(event->type)), z_pair(z_string("windowId"),z_int(event->window_id)),
        z_pair(z_string("timestamp"),z_int(event->timestamp)), z_pair(z_string("data"),event->data)
    };
    ZValue result=z_dict_from(pairs,4);
    if(event->data.tag==ZV_DICT){
        for(size_t i=0;i<event->data.as.dict->len;i++)z_set_index(result,event->data.as.dict->keys[i],event->data.as.dict->values[i]);
    }
    return result;
}
static void z_desktop_add_handler(ZDesktopHandlerNative **head, const char *name, ZValue handler) {
    ZDesktopHandlerNative *item=(ZDesktopHandlerNative *)z_alloc(sizeof(*item)); item->name=z_strdup(name); item->handler=handler; item->next=*head; *head=item;
}
static void z_desktop_normalize_shortcut(const char *input, char *output, size_t size) {
    size_t written=0; bool last_space=false;
    for(const unsigned char *p=(const unsigned char *)input; *p && written+1<size; p++){
        char c=(char)tolower(*p); if(isspace((unsigned char)c)){last_space=true;continue;} if(last_space&&c!='+'){last_space=false;} output[written++]=c;
    }
    output[written]='\0';
}
static ZDesktopWindowNative *z_desktop_find_window(ZDesktopAppNative *app, int64_t id) {
    for (ZDesktopWindowNative *window = app->windows; window != NULL; window = window->next) {
        if (window->id == id) return window;
    }
    return NULL;
}
static size_t z_desktop_open_windows(ZDesktopAppNative *app) {
    size_t count=0;for(ZDesktopWindowNative *window=app->windows;window!=NULL;window=window->next)if(window->open)count++;return count;
}
static void z_desktop_close_window_native(ZDesktopWindowNative *window) {
    if (window == NULL || !window->open) return;
    window->open = false;
#if defined(ZUMBRA_DESKTOP_SDL3)
    if(!window->app->headless&&window->handle!=NULL){if(window->frame_texture!=NULL&&z_desktop_sdl.DestroyTexture!=NULL){z_desktop_sdl.DestroyTexture((SDL_Texture*)window->frame_texture);window->frame_texture=NULL;}if(window->renderer!=NULL){z_ui_release_renderer_resources(window->renderer);z_desktop_sdl.DestroyRenderer((SDL_Renderer*)window->renderer);window->renderer=NULL;}z_desktop_sdl.DestroyWindow((SDL_Window *)window->handle);window->handle=NULL;}
#endif
    z_desktop_queue(window->app,"window_closed",window->id,z_dict_from(NULL,0));
}
static void z_desktop_dispatch(ZDesktopAppNative *app, ZDesktopEventNative *event) {
    if (event == NULL) return;
    z_ui_dispatch_from_desktop(app,event);
    ZValue event_value = z_desktop_event_value(event);
    for(ZDesktopHandlerNative *h=app->handlers;h!=NULL;h=h->next){if(strcmp(h->name,event->type)==0||strcmp(h->name,"event")==0){(void)z_call(h->handler,&event_value,1);}}
    if(strcmp(event->type,"key_down")==0){ZValue raw=z_desktop_dict_get(event->data,"shortcut");if(raw.tag==ZV_STRING){char normalized[256];z_desktop_normalize_shortcut(raw.as.s,normalized,sizeof(normalized));for(ZDesktopHandlerNative*h=app->shortcuts;h!=NULL;h=h->next)if(strcmp(h->name,normalized)==0)(void)z_call(h->handler,&event_value,1);}}
    if(strcmp(event->type,"tray")==0){ZValue id=z_desktop_dict_get(event->data,"id");if(id.tag==ZV_STRING){char key[320];snprintf(key,sizeof(key),"tray:%s",id.as.s);for(ZDesktopHandlerNative*h=app->handlers;h!=NULL;h=h->next)if(strcmp(h->name,key)==0)(void)z_call(h->handler,&event_value,1);}}
    if(strcmp(event->type,"window_close_requested")==0&&app->close_on_request)z_desktop_close_window_native(z_desktop_find_window(app,event->window_id));
    if((strcmp(event->type,"window_closed")==0)&&app->quit_on_last_window&&z_desktop_open_windows(app)==0)app->running=false;
    if(strcmp(event->type,"quit")==0||strcmp(event->type,"terminating")==0)app->running=false;
}

#if defined(ZUMBRA_DESKTOP_SDL3)
static void z_desktop_shortcut(uint32_t key,uint16_t mod,char *output,size_t size){
    output[0]='\0'; const char *key_name=z_desktop_sdl.GetKeyName(key); size_t used=0;
    if(mod&0x00c0u)used+=(size_t)snprintf(output+used,size-used,"ctrl+");
    if(mod&0x0003u)used+=(size_t)snprintf(output+used,size-used,"shift+");
    if(mod&0x0300u)used+=(size_t)snprintf(output+used,size-used,"alt+");
    if(mod&0x0c00u)used+=(size_t)snprintf(output+used,size-used,"super+");
    if(key_name!=NULL&&used<size){for(const unsigned char*p=(const unsigned char*)key_name;*p&&used+1<size;p++)output[used++]=(char)tolower(*p);output[used]='\0';}
}
static ZDesktopEventNative *z_desktop_sdl_event(int64_t timeout_ms) {
    ZDesktopSDLEvent raw; bool ok=timeout_ms<0?z_desktop_sdl.WaitEvent(&raw):(timeout_ms==0?z_desktop_sdl.PollEvent(&raw):z_desktop_sdl.WaitEventTimeout(&raw,(int32_t)timeout_ms)); if(!ok)return NULL;
    ZDesktopSDLCommonEvent *common=(ZDesktopSDLCommonEvent *)&raw; const char *type="unknown";int64_t window_id=0;ZValue data=z_dict_from(NULL,0);
    if(common->type==Z_DESKTOP_POLL_SENTINEL)type="__zumbra_poll_sentinel"; else if(common->type==Z_DESKTOP_WAKE_EVENT)type="__zumbra_wake"; else if(common->type==0x100u)type="quit"; else if(common->type==0x101u)type="terminating"; else if(common->type==0x102u)type="low_memory"; else if(common->type==0x103u)type="will_enter_background"; else if(common->type==0x104u)type="did_enter_background"; else if(common->type==0x105u)type="will_enter_foreground"; else if(common->type==0x106u)type="did_enter_foreground"; else if(common->type==0x109u)type="system_theme_changed";
    else if(common->type>=0x202u&&common->type<=0x220u){ZDesktopSDLWindowEvent*w=(ZDesktopSDLWindowEvent*)&raw;window_id=w->windowID;switch(common->type){case 0x202:type="window_shown";break;case 0x203:type="window_hidden";break;case 0x204:type="window_exposed";break;case 0x205:type="window_moved";data=z_desktop_dict2("x",z_int(w->data1),"y",z_int(w->data2));break;case 0x206:type="window_resized";data=z_desktop_dict2("width",z_int(w->data1),"height",z_int(w->data2));break;case 0x207:type="window_pixel_size_changed";data=z_desktop_dict2("width",z_int(w->data1),"height",z_int(w->data2));break;case 0x209:type="window_minimized";break;case 0x20a:type="window_maximized";break;case 0x20b:type="window_restored";break;case 0x20c:type="window_mouse_enter";break;case 0x20d:type="window_mouse_leave";break;case 0x20e:type="window_focus_gained";break;case 0x20f:type="window_focus_lost";break;case 0x210:type="window_close_requested";break;case 0x219:type="window_enter_fullscreen";break;case 0x21a:type="window_leave_fullscreen";break;case 0x21b:type="window_closed";break;default:type="window_event";break;}}
    else if(common->type==0x300u||common->type==0x301u){ZDesktopSDLKeyboardEvent*k=(ZDesktopSDLKeyboardEvent*)&raw;window_id=k->windowID;char shortcut[256];z_desktop_shortcut(k->key,k->mod,shortcut,sizeof(shortcut));const char*key=z_desktop_sdl.GetKeyName(k->key);ZValue pairs[5]={z_pair(z_string("key"),z_string(key==NULL?"":key)),z_pair(z_string("keycode"),z_int(k->key)),z_pair(z_string("modifiers"),z_int(k->mod)),z_pair(z_string("repeat"),z_bool(k->repeat)),z_pair(z_string("shortcut"),z_string(shortcut))};data=z_dict_from(pairs,5);type=common->type==0x300u?"key_down":"key_up";}
    else if(common->type==0x303u){ZDesktopSDLTextEvent*t=(ZDesktopSDLTextEvent*)&raw;window_id=t->windowID;data=z_desktop_dict1("text",z_string(t->text==NULL?"":t->text));type="text_input";}
    else if(common->type==0x400u){ZDesktopSDLMouseMotionEvent*m=(ZDesktopSDLMouseMotionEvent*)&raw;window_id=m->windowID;ZValue pairs[5]={z_pair(z_string("x"),z_float(m->x)),z_pair(z_string("y"),z_float(m->y)),z_pair(z_string("dx"),z_float(m->xrel)),z_pair(z_string("dy"),z_float(m->yrel)),z_pair(z_string("buttons"),z_int(m->state))};data=z_dict_from(pairs,5);type="mouse_move";}
    else if(common->type==0x401u||common->type==0x402u){ZDesktopSDLMouseButtonEvent*b=(ZDesktopSDLMouseButtonEvent*)&raw;window_id=b->windowID;ZValue pairs[4]={z_pair(z_string("button"),z_int(b->button)),z_pair(z_string("clicks"),z_int(b->clicks)),z_pair(z_string("x"),z_float(b->x)),z_pair(z_string("y"),z_float(b->y))};data=z_dict_from(pairs,4);type=common->type==0x401u?"mouse_down":"mouse_up";}
    else if(common->type==0x403u){ZDesktopSDLMouseWheelEvent*w=(ZDesktopSDLMouseWheelEvent*)&raw;window_id=w->windowID;ZValue pairs[4]={z_pair(z_string("x"),z_float(w->mouse_x)),z_pair(z_string("y"),z_float(w->mouse_y)),z_pair(z_string("dx"),z_float(w->x)),z_pair(z_string("dy"),z_float(w->y))};data=z_dict_from(pairs,4);type="mouse_wheel";}
    else if(common->type==0x900u)type="clipboard_updated";
    else if(common->type>=0x1000u&&common->type<=0x1004u){ZDesktopSDLDropEvent*d=(ZDesktopSDLDropEvent*)&raw;window_id=d->windowID;if(common->type==0x1000u){ZValue pairs[4]={z_pair(z_string("path"),z_string(d->data==NULL?"":d->data)),z_pair(z_string("source"),z_string(d->source==NULL?"":d->source)),z_pair(z_string("x"),z_float(d->x)),z_pair(z_string("y"),z_float(d->y))};data=z_dict_from(pairs,4);type="drop_file";}else if(common->type==0x1001u){data=z_desktop_dict1("text",z_string(d->data==NULL?"":d->data));type="drop_text";}else if(common->type==0x1002u)type="drop_begin";else if(common->type==0x1003u)type="drop_complete";}
    ZDesktopEventNative *event=(ZDesktopEventNative*)z_alloc(sizeof(*event));event->type=z_strdup(type);event->window_id=window_id;event->timestamp=(int64_t)common->timestamp;event->data=data;event->next=NULL;return event;
}
#endif
static ZDesktopEventNative *z_desktop_next_event(ZDesktopAppNative *app,int64_t timeout_ms){
    ZDesktopEventNative*queued=z_desktop_dequeue(app);if(queued!=NULL)return queued;
#if defined(ZUMBRA_DESKTOP_SDL3)
    if(!app->headless){
        for(;;){
            ZDesktopEventNative*event=z_desktop_sdl_event(timeout_ms);
            if(event==NULL)return NULL;
            if(strcmp(event->type,"__zumbra_poll_sentinel")==0){if(timeout_ms==0)return NULL;z_desktop_sleep_ms(1);continue;}
            if(strcmp(event->type,"__zumbra_wake")!=0)return event;
            queued=z_desktop_dequeue(app);if(queued!=NULL)return queued;
            if(timeout_ms==0)return NULL;
        }
    }
#endif
    z_desktop_sleep_ms(timeout_ms);return z_desktop_dequeue(app);
}

/* -------- integration helpers -------- */
#if defined(_WIN32)
static bool z_desktop_command_exists(const char *name){char found[PATH_MAX];return SearchPathA(NULL,name,NULL,(DWORD)sizeof(found),found,NULL)>0;}
static char *z_desktop_capture(const char *command){FILE*pipe=z_desktop_popen(command,"r");if(pipe==NULL)return NULL;char*result=(char*)malloc(8192);if(result==NULL){z_desktop_pclose(pipe);return NULL;}size_t n=fread(result,1,8191,pipe);int status=z_desktop_pclose(pipe);if(status!=0){free(result);return NULL;}while(n>0&&(result[n-1]=='\n'||result[n-1]=='\r'))n--;result[n]='\0';return result;}
static void z_desktop_spawn_detached(const char *command,const char *a,const char*b){char line[4096];snprintf(line,sizeof(line),"\"%s\"%s%s%s%s",command,a?" \"":"",a?a:"",a?"\"":"",b?"":"");if(b){size_t used=strlen(line);snprintf(line+used,sizeof(line)-used," \"%s\"",b);}STARTUPINFOA si;PROCESS_INFORMATION pi;memset(&si,0,sizeof(si));memset(&pi,0,sizeof(pi));si.cb=sizeof(si);if(CreateProcessA(NULL,line,NULL,NULL,FALSE,CREATE_NO_WINDOW|DETACHED_PROCESS,NULL,NULL,&si,&pi)){CloseHandle(pi.hThread);CloseHandle(pi.hProcess);}}
static ZValue z_desktop_pick_file_native(ZValue options,bool folder){const char*title=z_desktop_opt_string(options,"title",folder?"Choose folder":"Choose file");const char*default_path=z_desktop_opt_string(options,"defaultPath","");const char*mode=z_desktop_opt_string(options,"mode","open");bool save=z_desktop_opt_bool(options,"save",false)||strcasecmp(mode,"save")==0;if(folder){BROWSEINFOA browse;memset(&browse,0,sizeof(browse));browse.lpszTitle=title;browse.ulFlags=BIF_RETURNONLYFSDIRS|BIF_NEWDIALOGSTYLE;CoInitializeEx(NULL,COINIT_APARTMENTTHREADED);PIDLIST_ABSOLUTE item=SHBrowseForFolderA(&browse);char path[PATH_MAX];bool ok=item!=NULL&&SHGetPathFromIDListA(item,path);if(item)CoTaskMemFree(item);CoUninitialize();return ok?z_string(path):z_null();}char buffer[65536];memset(buffer,0,sizeof(buffer));if(*default_path)snprintf(buffer,sizeof(buffer),"%s",default_path);OPENFILENAMEA dialog;memset(&dialog,0,sizeof(dialog));dialog.lStructSize=sizeof(dialog);dialog.lpstrFile=buffer;dialog.nMaxFile=(DWORD)sizeof(buffer);dialog.lpstrTitle=title;dialog.lpstrFilter="All files\0*.*\0\0";dialog.Flags=OFN_EXPLORER|OFN_PATHMUSTEXIST;if(save){dialog.Flags|=OFN_OVERWRITEPROMPT;if(!GetSaveFileNameA(&dialog))return z_array_from(NULL,0);}else{dialog.Flags|=OFN_FILEMUSTEXIST;if(z_desktop_opt_bool(options,"multiple",false))dialog.Flags|=OFN_ALLOWMULTISELECT;if(!GetOpenFileNameA(&dialog))return z_array_from(NULL,0);}char*first=buffer;char*second=first+strlen(first)+1;if(save||*second=='\0'){ZValue one=z_string(first);return z_array_from(&one,1);}ZValue items[128];size_t count=0;for(char*name=second;*name&&count<128;name+=strlen(name)+1){char full[PATH_MAX];snprintf(full,sizeof(full),"%s\\%s",first,name);items[count++]=z_string(full);}return z_array_from(items,count);}
static ZValue z_desktop_paths_native(void){char home[PATH_MAX]="",cwd[PATH_MAX]="",exe[PATH_MAX]="",temp[PATH_MAX]="";DWORD n=GetEnvironmentVariableA("USERPROFILE",home,(DWORD)sizeof(home));if(n==0)GetEnvironmentVariableA("HOMEPATH",home,(DWORD)sizeof(home));GetCurrentDirectoryA((DWORD)sizeof(cwd),cwd);GetModuleFileNameA(NULL,exe,(DWORD)sizeof(exe));GetTempPathA((DWORD)sizeof(temp),temp);const char*cache=getenv("LOCALAPPDATA");const char*config=getenv("APPDATA");const char*data=getenv("APPDATA");ZValue pairs[8]={z_pair(z_string("home"),z_string(home)),z_pair(z_string("cache"),z_string(cache?cache:"")),z_pair(z_string("config"),z_string(config?config:"")),z_pair(z_string("data"),z_string(data?data:"")),z_pair(z_string("runtime"),z_string(temp)),z_pair(z_string("executable"),z_string(exe)),z_pair(z_string("cwd"),z_string(cwd)),z_pair(z_string("temp"),z_string(temp))};return z_dict_from(pairs,8);}
static bool z_desktop_notify_native(const char*title,const char*body){return MessageBoxA(NULL,body,title,MB_OK|MB_ICONINFORMATION|MB_SETFOREGROUND)!=0;}
static bool z_desktop_open_external_native(const char*target){HINSTANCE result=ShellExecuteA(NULL,"open",target,NULL,NULL,SW_SHOWNORMAL);return (INT_PTR)result>32;}
static char *z_desktop_windows_quote(const char*value){size_t n=strlen(value);char*out=(char*)malloc(n*2+3);if(!out)z_fatal("out of memory");size_t j=0;out[j++]='\"';size_t slashes=0;for(size_t i=0;i<n;i++){char c=value[i];if(c=='\\'){slashes++;continue;}if(c=='\"'){for(size_t k=0;k<slashes*2+1;k++)out[j++]='\\';out[j++]='\"';slashes=0;continue;}while(slashes){out[j++]='\\';slashes--;}out[j++]=c;}while(slashes){out[j++]='\\';out[j++]='\\';slashes--;}out[j++]='\"';out[j]='\0';return out;}
static ZValue z_desktop_process_spawn(const char*command,ZValue options){if(command==NULL||*command=='\0')z_fatal("desktopSpawn command cannot be empty");ZValue arg_value=z_desktop_dict_get(options,"args");size_t extra=arg_value.tag==ZV_ARRAY?arg_value.as.array->len:0;char*quoted=z_desktop_windows_quote(command);size_t size=strlen(quoted)+1;char**args=extra?calloc(extra,sizeof(char*)):NULL;for(size_t i=0;i<extra;i++){ZValue value=arg_value.as.array->items[i];if(value.tag!=ZV_STRING)z_fatal("desktopSpawn args must contain strings");args[i]=z_desktop_windows_quote(value.as.s);size+=strlen(args[i])+1;}char*line=calloc(size+1,1);if(!line)z_fatal("out of memory");strcpy(line,quoted);free(quoted);for(size_t i=0;i<extra;i++){strcat(line," ");strcat(line,args[i]);free(args[i]);}free(args);STARTUPINFOA si;PROCESS_INFORMATION pi;memset(&si,0,sizeof(si));memset(&pi,0,sizeof(pi));si.cb=sizeof(si);const char*cwd=z_desktop_opt_string(options,"cwd","");BOOL ok=CreateProcessA(NULL,line,NULL,NULL,FALSE,0,NULL,*cwd?cwd:NULL,&si,&pi);free(line);if(!ok)z_fatal("desktopSpawn failed: Windows error %lu",(unsigned long)GetLastError());CloseHandle(pi.hThread);ZDesktopProcessNative*p=(ZDesktopProcessNative*)z_alloc(sizeof(*p));p->handle=pi.hProcess;p->pid=pi.dwProcessId;p->running=true;p->waited=false;p->exit_code=0;return z_desktop_object(ZV_DESKTOP_PROCESS,ZK_DESKTOP_PROCESS,p);}
#else
static char *z_desktop_shell_quote(const char *value){size_t n=3;for(const char*p=value;*p;p++)n+=(*p=='\''?4:1);char*out=(char*)malloc(n);if(out==NULL)z_fatal("out of memory");size_t i=0;out[i++]='\'';for(const char*p=value;*p;p++){if(*p=='\''){memcpy(out+i,"'\\''",4);i+=4;}else out[i++]=*p;}out[i++]='\'';out[i]='\0';return out;}
#if defined(__APPLE__)
static char *z_desktop_applescript_string(const char *value){size_t n=3;for(const char*p=value;*p;p++)n+=(*p=='\\'||*p=='"'?2:1);char*out=(char*)malloc(n);if(out==NULL)z_fatal("out of memory");size_t i=0;out[i++]='"';for(const char*p=value;*p;p++){if(*p=='\\'||*p=='"')out[i++]='\\';out[i++]=*p;}out[i++]='"';out[i]='\0';return out;}
#endif
static char *z_desktop_capture(const char *command){FILE*pipe=z_desktop_popen(command,"r");if(pipe==NULL)return NULL;char*result=(char*)malloc(8192);if(result==NULL){z_desktop_pclose(pipe);return NULL;}size_t n=fread(result,1,8191,pipe);int status=z_desktop_pclose(pipe);if(status!=0){free(result);return NULL;}while(n>0&&(result[n-1]=='\n'||result[n-1]=='\r'))n--;result[n]='\0';return result;}
static bool z_desktop_command_exists(const char *name){char command[512];snprintf(command,sizeof(command),"command -v %s >/dev/null 2>&1",name);return system(command)==0;}
static void z_desktop_spawn_detached(const char *command,const char *a,const char*b){pid_t pid=fork();if(pid<0)return;if(pid==0){pid_t child=fork();if(child<0)_exit(127);if(child>0)_exit(0);setsid();if(b!=NULL)execlp(command,command,a,b,(char*)NULL);else if(a!=NULL)execlp(command,command,a,(char*)NULL);else execlp(command,command,(char*)NULL);_exit(127);}int status=0;while(waitpid(pid,&status,0)<0&&errno==EINTR){}}
static ZValue z_desktop_pick_file_native(ZValue options,bool folder){const char*default_path=z_desktop_opt_string(options,"defaultPath","");const char*title=z_desktop_opt_string(options,"title",folder?"Choose folder":"Choose file");const char*mode=z_desktop_opt_string(options,"mode","open");bool save=z_desktop_opt_bool(options,"save",false)||strcasecmp(mode,"save")==0;bool multiple=!save&&z_desktop_opt_bool(options,"multiple",false);char command[4096];
#if defined(__APPLE__)
    char script[3072];char*at=z_desktop_applescript_string(title);char*ap=z_desktop_applescript_string(default_path);if(folder){if(*default_path)snprintf(script,sizeof(script),"POSIX path of (choose folder with prompt %s default location POSIX file %s)",at,ap);else snprintf(script,sizeof(script),"POSIX path of (choose folder with prompt %s)",at);}else if(save){if(*default_path)snprintf(script,sizeof(script),"POSIX path of (choose file name with prompt %s default location POSIX file %s)",at,ap);else snprintf(script,sizeof(script),"POSIX path of (choose file name with prompt %s)",at);}else if(multiple){snprintf(script,sizeof(script),"set xs to choose file with prompt %s with multiple selections allowed\nset AppleScript's text item delimiters to linefeed\nset ys to {}\nrepeat with x in xs\nset end of ys to POSIX path of x\nend repeat\nys as text",at);}else{if(*default_path)snprintf(script,sizeof(script),"POSIX path of (choose file with prompt %s default location POSIX file %s)",at,ap);else snprintf(script,sizeof(script),"POSIX path of (choose file with prompt %s)",at);}char*quoted=z_desktop_shell_quote(script);snprintf(command,sizeof(command),"osascript -e %s",quoted);free(quoted);free(at);free(ap);
#else
    char*qp=z_desktop_shell_quote(default_path);char*qt=z_desktop_shell_quote(title);if(z_desktop_command_exists("zenity")){snprintf(command,sizeof(command),"zenity --file-selection %s %s %s",folder?"--directory":"",save?"--save --confirm-overwrite":"",multiple&&!folder?"--multiple --separator='\\n'":"");if(*title){size_t used=strlen(command);snprintf(command+used,sizeof(command)-used," --title=%s",qt);}if(*default_path){size_t used=strlen(command);snprintf(command+used,sizeof(command)-used," --filename=%s",qp);}}else if(z_desktop_command_exists("kdialog")){snprintf(command,sizeof(command),"kdialog %s %s",folder?"--getexistingdirectory":save?"--getsavefilename":"--getopenfilename",qp);}else{free(qp);free(qt);return folder?z_null():z_array_from(NULL,0);}free(qp);free(qt);
#endif
    char*out=z_desktop_capture(command);if(out==NULL||*out=='\0'){free(out);return folder?z_null():z_array_from(NULL,0);}if(folder){ZValue result=z_string(out);free(out);return result;}ZValue items[128];size_t count=0;char*token_state=NULL;for(char*line=strtok_r(out,"\n",&token_state);line!=NULL&&count<128;line=strtok_r(NULL,"\n",&token_state))items[count++]=z_string(line);free(out);return z_array_from(items,count);}
static ZValue z_desktop_paths_native(void){char home[PATH_MAX]="",cwd[PATH_MAX]="",exe[PATH_MAX]="";const char*h=getenv("HOME");if(h!=NULL)snprintf(home,sizeof(home),"%s",h);if(z_desktop_getcwd(cwd,sizeof(cwd))==NULL)cwd[0]='\0';
#if defined(__APPLE__)
    extern int _NSGetExecutablePath(char*,uint32_t*);uint32_t length=(uint32_t)sizeof(exe);if(_NSGetExecutablePath(exe,&length)!=0)exe[0]='\0';
#else
    ssize_t n=readlink("/proc/self/exe",exe,sizeof(exe)-1);if(n>=0)exe[n]='\0';else exe[0]='\0';
#endif
    const char*cache=getenv("XDG_CACHE_HOME");const char*config=getenv("XDG_CONFIG_HOME");const char*data=getenv("XDG_DATA_HOME");const char*runtime_dir=getenv("XDG_RUNTIME_DIR");char cache_buf[PATH_MAX],config_buf[PATH_MAX],data_buf[PATH_MAX];
#if defined(__APPLE__)
    if(cache==NULL){snprintf(cache_buf,sizeof(cache_buf),"%s/Library/Caches",home);cache=cache_buf;}if(config==NULL){snprintf(config_buf,sizeof(config_buf),"%s/Library/Preferences",home);config=config_buf;}if(data==NULL){snprintf(data_buf,sizeof(data_buf),"%s/Library/Application Support",home);data=data_buf;}
#else
    if(cache==NULL){snprintf(cache_buf,sizeof(cache_buf),"%s/.cache",home);cache=cache_buf;}if(config==NULL){snprintf(config_buf,sizeof(config_buf),"%s/.config",home);config=config_buf;}if(data==NULL){snprintf(data_buf,sizeof(data_buf),"%s/.local/share",home);data=data_buf;}
#endif
    ZValue pairs[8]={z_pair(z_string("home"),z_string(home)),z_pair(z_string("cache"),z_string(cache)),z_pair(z_string("config"),z_string(config)),z_pair(z_string("data"),z_string(data)),z_pair(z_string("runtime"),z_string(runtime_dir==NULL?"":runtime_dir)),z_pair(z_string("executable"),z_string(exe)),z_pair(z_string("cwd"),z_string(cwd)),z_pair(z_string("temp"),z_string(getenv("TMPDIR")?getenv("TMPDIR"):"/tmp"))};return z_dict_from(pairs,8);}
static bool z_desktop_notify_native(const char*title,const char*body){
#if defined(__APPLE__)
    char*at=z_desktop_applescript_string(title);char*ab=z_desktop_applescript_string(body);char script[3072];snprintf(script,sizeof(script),"display notification %s with title %s",ab,at);char*quoted=z_desktop_shell_quote(script);char command[4096];snprintf(command,sizeof(command),"osascript -e %s",quoted);free(at);free(ab);free(quoted);return system(command)==0;
#else
    if (!z_desktop_command_exists("notify-send")) return false;
    z_desktop_spawn_detached("notify-send", title, body);
    return true;
#endif
}
static bool z_desktop_open_external_native(const char*target){
#if defined(__APPLE__)
    if(!z_desktop_command_exists("open"))return false;z_desktop_spawn_detached("open",target,NULL);return true;
#else
    if (!z_desktop_command_exists("xdg-open")) return false;
    z_desktop_spawn_detached("xdg-open", target, NULL);
    return true;
#endif
}
static ZValue z_desktop_process_spawn(const char*command,ZValue options){if(command==NULL||*command=='\0')z_fatal("desktopSpawn command cannot be empty");ZValue arg_value=z_desktop_dict_get(options,"args");size_t extra=(arg_value.tag==ZV_ARRAY?arg_value.as.array->len:0);char**argv=(char**)calloc(extra+2,sizeof(char*));if(argv==NULL)z_fatal("out of memory");argv[0]=(char*)command;for(size_t i=0;i<extra;i++){ZValue value=arg_value.as.array->items[i];if(value.tag!=ZV_STRING)z_fatal("desktopSpawn args must contain strings");argv[i+1]=(char*)value.as.s;}argv[extra+1]=NULL;pid_t pid=fork();if(pid<0){free(argv);z_fatal("desktopSpawn failed: %s",strerror(errno));}if(pid==0){const char*cwd=z_desktop_opt_string(options,"cwd","");if(*cwd&&z_desktop_chdir(cwd)!=0)_exit(126);ZValue env=z_desktop_dict_get(options,"env");if(env.tag==ZV_DICT){for(size_t i=0;i<env.as.dict->len;i++){ZValue k=env.as.dict->keys[i],v=env.as.dict->values[i];if(k.tag==ZV_STRING&&v.tag==ZV_STRING)setenv(k.as.s,v.as.s,1);}}execvp(command,argv);_exit(127);}free(argv);ZDesktopProcessNative*p=(ZDesktopProcessNative*)z_alloc(sizeof(*p));p->pid=pid;p->running=true;p->waited=false;p->exit_code=0;return z_desktop_object(ZV_DESKTOP_PROCESS,ZK_DESKTOP_PROCESS,p);}
#endif

/* -------- app and window primitives -------- */
static ZValue z_desktop_app_new(ZValue options){
    if (options.tag != ZV_DICT) z_fatal("desktopApp expects options dictionary");
    ZDesktopAppNative*app=(ZDesktopAppNative*)z_alloc(sizeof(*app));memset(app,0,sizeof(*app));pthread_mutex_init(&app->mutex,NULL);app->close_on_request=z_desktop_opt_bool(options,"closeOnRequest",true);app->quit_on_last_window=z_desktop_opt_bool(options,"quitOnLastWindow",true);app->poll_interval_ms=z_desktop_opt_int(options,"pollIntervalMs",16);app->next_window_id=1;const char*backend=z_desktop_opt_string(options,"backend","auto");const char*headless_env=getenv("ZUMBRA_DESKTOP_HEADLESS");app->headless=strcmp(backend,"headless")==0||(headless_env!=NULL&&(strcmp(headless_env,"1")==0||strcasecmp(headless_env,"true")==0));
#if defined(ZUMBRA_DESKTOP_SDL3)
    if(!app->headless){if(!z_desktop_sdl_load())z_fatal("desktopApp: %s",z_desktop_sdl.error);if(z_desktop_sdl.SetAppMetadata!=NULL)z_desktop_sdl.SetAppMetadata(z_desktop_opt_string(options,"name","Zumbra"),z_desktop_opt_string(options,"version",""),z_desktop_opt_string(options,"identifier","dev.zumbra.app"));}
#endif
    if (app->headless) z_desktop_queue(app,"ready",0,z_desktop_dict1("backend",z_string("headless")));
    return z_desktop_object(ZV_DESKTOP_APP,ZK_DESKTOP_APP,app);
}
static ZValue z_desktop_window_new(ZDesktopAppNative*app,ZValue options){
    if (options.tag != ZV_DICT) z_fatal("desktopWindow options must be dictionary");
    if (app->closed) z_fatal("desktop app is closed");
    int64_t width=z_desktop_opt_int(options,"width",800),height=z_desktop_opt_int(options,"height",600);if(width<1||height<1)z_fatal("window width and height must be positive");ZDesktopWindowNative*w=(ZDesktopWindowNative*)z_alloc(sizeof(*w));memset(w,0,sizeof(*w));w->app=app;w->title=z_strdup(z_desktop_opt_string(options,"title","Zumbra"));w->width=width;w->height=height;w->pixel_width=width;w->pixel_height=height;w->scale=z_desktop_opt_float(options,"scale",1.0);if(w->scale<=0)w->scale=1.0;w->density=z_desktop_opt_bool(options,"highDPI",true)?w->scale:1.0;w->pixel_width=(int64_t)((double)width*w->density);w->pixel_height=(int64_t)((double)height*w->density);w->open=true;w->visible=!z_desktop_opt_bool(options,"hidden",false);w->fullscreen=z_desktop_opt_bool(options,"fullscreen",false);
#if defined(ZUMBRA_DESKTOP_SDL3)
    if(!app->headless){uint64_t flags=0;if(w->fullscreen)flags|=0x1u;if(!w->visible)flags|=0x8u;if(z_desktop_opt_bool(options,"borderless",false))flags|=0x10u;if(z_desktop_opt_bool(options,"resizable",true))flags|=0x20u;if(z_desktop_opt_bool(options,"highDPI",true))flags|=0x2000u;if(z_desktop_opt_bool(options,"alwaysOnTop",false))flags|=0x8000u;if(z_desktop_opt_bool(options,"transparent",false))flags|=0x40000000u;SDL_Window*handle=z_desktop_sdl.CreateWindow(w->title,(int)width,(int)height,flags);if(handle==NULL)z_fatal("desktopWindow: %s",z_desktop_sdl.GetError());w->handle=handle;w->id=(int64_t)z_desktop_sdl.GetWindowID(handle);w->scale=z_desktop_sdl.GetWindowDisplayScale(handle);w->density=z_desktop_sdl.GetWindowPixelDensity(handle);int pixel_width=(int)width,pixel_height=(int)height;if(z_desktop_sdl.GetWindowSizeInPixels!=NULL&&z_desktop_sdl.GetWindowSizeInPixels(handle,&pixel_width,&pixel_height)){w->pixel_width=pixel_width;w->pixel_height=pixel_height;}const char*icon=z_desktop_opt_string(options,"icon","");if(*icon){SDL_Surface*surface=z_desktop_sdl_load_bmp(icon);if(surface==NULL)z_fatal("could not load BMP icon");if(!z_desktop_sdl.SetWindowIcon(handle,surface)){z_desktop_sdl.DestroySurface(surface);z_fatal("could not set window icon");}z_desktop_sdl.DestroySurface(surface);}}
#endif
    if (app->headless) w->id = app->next_window_id++;
    w->next = app->windows;
    app->windows = w;
    if (app->headless) {if(w->visible)z_desktop_queue(app,"window_shown",w->id,z_dict_from(NULL,0));z_desktop_queue(app,"window_resized",w->id,z_desktop_dict2("width",z_int(width),"height",z_int(height)));z_desktop_queue(app,"window_pixel_size_changed",w->id,z_desktop_dict3("width",z_int(w->pixel_width),"height",z_int(w->pixel_height),"scale",z_float(w->scale)));}return z_desktop_object(ZV_DESKTOP_WINDOW,ZK_DESKTOP_WINDOW,w);
}
static ZValue z_desktop_window_action(ZDesktopWindowNative*w,const char*name){if(!w->open)z_fatal("window is closed");
#if defined(ZUMBRA_DESKTOP_SDL3)
    if(!w->app->headless){bool ok=true;if(strcmp(name,"show")==0)ok=z_desktop_sdl.ShowWindow((SDL_Window*)w->handle);else if(strcmp(name,"hide")==0)ok=z_desktop_sdl.HideWindow((SDL_Window*)w->handle);else if(strcmp(name,"maximize")==0)ok=z_desktop_sdl.MaximizeWindow((SDL_Window*)w->handle);else if(strcmp(name,"minimize")==0)ok=z_desktop_sdl.MinimizeWindow((SDL_Window*)w->handle);else if(strcmp(name,"restore")==0)ok=z_desktop_sdl.RestoreWindow((SDL_Window*)w->handle);else if(strcmp(name,"focus")==0)ok=z_desktop_sdl.RaiseWindow((SDL_Window*)w->handle);if(!ok)z_fatal("desktop window action failed: %s",z_desktop_sdl.GetError());}
#endif
    if(w->app->headless){const char*event=strcmp(name,"show")==0?"window_shown":strcmp(name,"hide")==0?"window_hidden":strcmp(name,"maximize")==0?"window_maximized":strcmp(name,"minimize")==0?"window_minimized":strcmp(name,"restore")==0?"window_restored":"window_focus_gained";z_desktop_queue(w->app,event,w->id,z_dict_from(NULL,0));}return z_desktop_object(ZV_DESKTOP_WINDOW,ZK_DESKTOP_WINDOW,w);}


/* -------- tray callback -------- */
#if defined(ZUMBRA_DESKTOP_SDL3)
typedef struct { ZDesktopAppNative*app; char*id; } ZDesktopTrayCallback;
static void z_desktop_tray_callback(void*userdata,SDL_TrayEntry*entry){(void)entry;ZDesktopTrayCallback*callback=(ZDesktopTrayCallback*)userdata;if(callback!=NULL)z_desktop_queue(callback->app,"tray",0,z_desktop_dict1("id",z_string(callback->id)));}
#endif

static ZValue z_desktop_method(ZValue receiver,const char*builtin){ZDesktopMethodNative*m=(ZDesktopMethodNative*)z_alloc(sizeof(*m));m->receiver=receiver;m->builtin=builtin;return z_desktop_object(ZV_DESKTOP_METHOD,ZK_FUNCTION,m);}
static ZValue z_desktop_get_field(ZValue value,const char*name,bool*handled){*handled=true;
#define ZD_METHOD(prop,builtin) if(strcmp(name,prop)==0)return z_desktop_method(value,builtin)
    if(value.tag==ZV_DESKTOP_APP){ZD_METHOD("backend","desktopBackend");ZD_METHOD("window","desktopWindow");ZD_METHOD("on","desktopOn");ZD_METHOD("shortcut","desktopShortcut");ZD_METHOD("poll","desktopPoll");ZD_METHOD("run","desktopRun");ZD_METHOD("quit","desktopQuit");ZD_METHOD("running","desktopRunning");ZD_METHOD("close","desktopClose");ZD_METHOD("emit","desktopEmit");ZD_METHOD("setClipboard","desktopSetClipboard");ZD_METHOD("clipboard","desktopClipboard");ZD_METHOD("pickFile","desktopPickFile");ZD_METHOD("pickFolder","desktopPickFolder");ZD_METHOD("notify","desktopNotify");ZD_METHOD("paths","desktopPaths");ZD_METHOD("openExternal","desktopOpenExternal");ZD_METHOD("tray","desktopTray");}
    else if(value.tag==ZV_DESKTOP_WINDOW){ZD_METHOD("show","desktopWindowShow");ZD_METHOD("hide","desktopWindowHide");ZD_METHOD("close","desktopWindowClose");ZD_METHOD("isOpen","desktopWindowOpen");ZD_METHOD("id","desktopWindowId");ZD_METHOD("title","desktopWindowTitle");ZD_METHOD("setTitle","desktopWindowSetTitle");ZD_METHOD("size","desktopWindowSize");ZD_METHOD("pixelSize","desktopWindowPixelSize");ZD_METHOD("setSize","desktopWindowSetSize");ZD_METHOD("position","desktopWindowPosition");ZD_METHOD("setPosition","desktopWindowSetPosition");ZD_METHOD("fullscreen","desktopWindowFullscreen");ZD_METHOD("setFullscreen","desktopWindowSetFullscreen");ZD_METHOD("maximize","desktopWindowMaximize");ZD_METHOD("minimize","desktopWindowMinimize");ZD_METHOD("restore","desktopWindowRestore");ZD_METHOD("focus","desktopWindowFocus");ZD_METHOD("displayScale","desktopWindowDisplayScale");ZD_METHOD("pixelDensity","desktopWindowPixelDensity");ZD_METHOD("setIcon","desktopWindowSetIcon");}
    else if(value.tag==ZV_DESKTOP_TRAY){ZD_METHOD("add","desktopTrayAdd");ZD_METHOD("setTooltip","desktopTrayTooltip");ZD_METHOD("close","desktopTrayClose");ZD_METHOD("isOpen","desktopTrayOpen");}
    else if(value.tag==ZV_DESKTOP_PROCESS){ZD_METHOD("wait","desktopProcessWait");ZD_METHOD("kill","desktopProcessKill");ZD_METHOD("running","desktopProcessRunning");ZD_METHOD("id","desktopProcessId");}
    else {*handled=false;return z_null();}
    z_fatal("desktop object has no method named %s",name);return z_null();
#undef ZD_METHOD
}
static ZValue z_desktop_call_method(ZValue callable,const ZValue*args,size_t argc){ZDesktopMethodNative*m=(ZDesktopMethodNative*)callable.as.p;ZValue*full=(ZValue*)z_alloc(sizeof(ZValue)*(argc+1));full[0]=m->receiver;if(argc)memcpy(full+1,args,sizeof(ZValue)*argc);return z_desktop_call_builtin(m->builtin,full,argc+1,NULL);}

static ZValue z_desktop_call_builtin(const char*name,const ZValue*args,size_t argc,bool*handled){if(handled!=NULL)*handled=true;
    if(strcmp(name,"desktopApp")==0){z_expect_args(name,argc,1);return z_desktop_app_new(args[0]);}
    if(strcmp(name,"desktopBackend")==0){z_expect_args(name,argc,1);return z_string(z_desktop_expect_app(args[0])->headless?"headless":"sdl3");}
    if(strcmp(name,"desktopWindow")==0){z_expect_args(name,argc,2);return z_desktop_window_new(z_desktop_expect_app(args[0]),args[1]);}
    if(strcmp(name,"desktopOn")==0){z_expect_args(name,argc,3);ZDesktopAppNative*app=z_desktop_expect_app(args[0]);z_desktop_add_handler(&app->handlers,z_as_cstring(args[1]),args[2]);return args[0];}
    if(strcmp(name,"desktopShortcut")==0){z_expect_args(name,argc,3);ZDesktopAppNative*app=z_desktop_expect_app(args[0]);char normalized[256];z_desktop_normalize_shortcut(z_as_cstring(args[1]),normalized,sizeof(normalized));z_desktop_add_handler(&app->shortcuts,normalized,args[2]);return args[0];}
    if(strcmp(name,"desktopPoll")==0){z_expect_args(name,argc,2);ZDesktopAppNative*app=z_desktop_expect_app(args[0]);
#if defined(ZUMBRA_DESKTOP_SDL3)
        z_desktop_refresh_gamepads(app);
#endif
        ZDesktopEventNative*event=z_desktop_next_event(app,z_as_i64(args[1]));if(event==NULL){z_ui_render_all(app);return z_null();}z_desktop_dispatch(app,event);z_ui_render_all(app);return z_desktop_event_value(event);}
    if(strcmp(name,"desktopRun")==0){z_expect_args(name,argc,1);ZDesktopAppNative*app=z_desktop_expect_app(args[0]);if(app->closed)z_fatal("desktop app is closed");app->running=true;app->run_thread=pthread_self();app->run_thread_set=true;z_ui_render_all(app);while(app->running&&!app->closed){if(z_ui_any_dirty(app))z_ui_render_all(app);
#if defined(ZUMBRA_DESKTOP_SDL3)
        z_desktop_refresh_gamepads(app);
#endif
        int64_t wait_ms=app->headless?app->poll_interval_ms:-1;ZDesktopEventNative*event=z_desktop_next_event(app,wait_ms);if(event!=NULL)z_desktop_dispatch(app,event);size_t drained=0;while(app->running&&!app->closed&&drained<256){ZDesktopEventNative*pending=z_desktop_dequeue(app);if(pending==NULL)break;z_desktop_dispatch(app,pending);drained++;}z_ui_render_all(app);}app->run_thread_set=false;return z_bool(true);}
    if(strcmp(name,"desktopQuit")==0){z_expect_args(name,argc,1);z_desktop_expect_app(args[0])->running=false;return z_bool(true);}
    if(strcmp(name,"desktopRunning")==0){z_expect_args(name,argc,1);return z_bool(z_desktop_expect_app(args[0])->running);}
    if(strcmp(name,"desktopClose")==0){z_expect_args(name,argc,1);ZDesktopAppNative*app=z_desktop_expect_app(args[0]);if(!app->closed){app->running=false;app->closed=true;for(ZDesktopWindowNative*w=app->windows;w!=NULL;w=w->next)z_desktop_close_window_native(w);for(ZDesktopTrayNative*t=app->trays;t!=NULL;t=t->next){if(t->open){
#if defined(ZUMBRA_DESKTOP_SDL3)
            if(!app->headless&&t->handle!=NULL&&z_desktop_sdl.DestroyTray!=NULL)z_desktop_sdl.DestroyTray((SDL_Tray*)t->handle);
#endif
            t->open=false;}}
#if defined(ZUMBRA_DESKTOP_SDL3)
        if(!app->headless){if(app->audio_stream!=NULL&&z_desktop_sdl.DestroyAudioStream!=NULL){z_desktop_sdl.DestroyAudioStream((SDL_AudioStream*)app->audio_stream);app->audio_stream=NULL;}for(int slot=0;slot<2;slot++){if(app->gamepads[slot]!=NULL&&z_desktop_sdl.CloseGamepad!=NULL)z_desktop_sdl.CloseGamepad((SDL_Gamepad*)app->gamepads[slot]);app->gamepads[slot]=NULL;app->gamepad_ids[slot]=0;}}
        if(!app->headless&&z_desktop_sdl.initialized){z_desktop_destroy_system_cursors();z_desktop_sdl.Quit();z_desktop_sdl.initialized=false;}
#endif
        }return z_bool(true);}
    if(strcmp(name,"desktopKeyDown")==0){z_expect_args(name,argc,2);ZDesktopAppNative*app=z_desktop_expect_app(args[0]);int64_t scancode=z_as_i64(args[1]);if(app->headless||scancode<0)return z_bool(false);
#if defined(ZUMBRA_DESKTOP_SDL3)
        if(z_desktop_sdl.GetKeyboardState==NULL)return z_bool(false);int count=0;const bool*keys=z_desktop_sdl.GetKeyboardState(&count);return z_bool(keys!=NULL&&scancode<count&&keys[scancode]);
#else
        return z_bool(false);
#endif
    }
    if(strcmp(name,"desktopGamepadButton")==0){z_expect_args(name,argc,3);ZDesktopAppNative*app=z_desktop_expect_app(args[0]);int64_t player=z_as_i64(args[1]),button=z_as_i64(args[2]);if(app->headless||player<1||player>2||button<0)return z_bool(false);
#if defined(ZUMBRA_DESKTOP_SDL3)
        z_desktop_refresh_gamepads(app);SDL_Gamepad*pad=(SDL_Gamepad*)app->gamepads[player-1];return z_bool(pad!=NULL&&z_desktop_sdl.GetGamepadButton!=NULL&&z_desktop_sdl.GetGamepadButton(pad,(int)button));
#else
        return z_bool(false);
#endif
    }
    if(strcmp(name,"desktopAudioQueue")==0){z_expect_args(name,argc,4);ZDesktopAppNative*app=z_desktop_expect_app(args[0]);ZValue samples=args[1];int64_t volume=z_as_i64(args[2]);bool muted=z_as_bool(args[3]);if(samples.tag!=ZV_BUFFER||samples.kind!=ZK_BYTE_ARRAY||samples.as.buffer==NULL)z_fatal("desktopAudioQueue expects ByteArray samples");size_t count=samples.as.buffer->len;if(app->headless||count==0)return z_int((int64_t)count);
#if defined(ZUMBRA_DESKTOP_SDL3)
        SDL_AudioStream*stream=z_desktop_audio_stream(app);if(stream==NULL||z_desktop_sdl.PutAudioStreamData==NULL)return z_int(0);if(volume<0)volume=0;if(volume>100)volume=100;int16_t*converted=(int16_t*)malloc(count*sizeof(int16_t));if(converted==NULL)z_fatal("desktopAudioQueue out of memory");const uint8_t*source=(const uint8_t*)samples.as.buffer->data;int64_t effective=muted?0:volume;for(size_t i=0;i<count;i++){int centered=((int)source[i]-32)*1024;converted[i]=(int16_t)((centered*effective)/100);}bool ok=z_desktop_sdl.PutAudioStreamData(stream,converted,(int)(count*sizeof(int16_t)));free(converted);return z_int(ok?(int64_t)count:0);
#else
        return z_int(0);
#endif
    }
    if(strcmp(name,"desktopAudioQueued")==0){z_expect_args(name,argc,1);ZDesktopAppNative*app=z_desktop_expect_app(args[0]);if(app->headless)return z_int(0);
#if defined(ZUMBRA_DESKTOP_SDL3)
        SDL_AudioStream*stream=z_desktop_audio_stream(app);return z_int(stream!=NULL&&z_desktop_sdl.GetAudioStreamQueued!=NULL?(int64_t)z_desktop_sdl.GetAudioStreamQueued(stream):0);
#else
        return z_int(0);
#endif
    }
    if(strcmp(name,"desktopWindowPresentRGBA")==0){z_expect_args(name,argc,4);ZDesktopWindowNative*w=z_desktop_expect_window(args[0]);ZValue pixels=args[1];int64_t width=z_as_i64(args[2]),height=z_as_i64(args[3]);if(!w->open)z_fatal("window is closed");if(pixels.tag!=ZV_BUFFER||pixels.kind!=ZK_BYTE_ARRAY||pixels.as.buffer==NULL)z_fatal("desktopWindowPresentRGBA expects ByteArray pixels");if(width<1||height<1||(uint64_t)width>(uint64_t)SIZE_MAX/4u/(uint64_t)height||pixels.as.buffer->len!=(size_t)(width*height*4))z_fatal("desktopWindowPresentRGBA expects width*height*4 bytes");if(w->app->headless)return z_bool(true);
#if defined(ZUMBRA_DESKTOP_SDL3)
        if(z_desktop_sdl.CreateTexture==NULL||z_desktop_sdl.UpdateTexture==NULL||z_desktop_sdl.RenderTexture==NULL)z_fatal("SDL3 streaming texture APIs are unavailable");if(w->renderer==NULL){w->renderer=z_desktop_sdl.CreateRenderer((SDL_Window*)w->handle,NULL);if(w->renderer==NULL)z_fatal("desktopWindowPresentRGBA renderer: %s",z_desktop_sdl.GetError());if(z_desktop_sdl.SetRenderLogicalPresentation!=NULL)(void)z_desktop_sdl.SetRenderLogicalPresentation((SDL_Renderer*)w->renderer,(int)width,(int)height,4);if(z_desktop_sdl.SetRenderVSync!=NULL)(void)z_desktop_sdl.SetRenderVSync((SDL_Renderer*)w->renderer,w->vsync?1:0);}if(w->frame_texture==NULL||w->frame_width!=width||w->frame_height!=height){if(w->frame_texture!=NULL&&z_desktop_sdl.DestroyTexture!=NULL)z_desktop_sdl.DestroyTexture((SDL_Texture*)w->frame_texture);w->frame_texture=z_desktop_sdl.CreateTexture((SDL_Renderer*)w->renderer,UINT32_C(0x16762004),1,(int)width,(int)height);w->frame_width=width;w->frame_height=height;if(w->frame_texture==NULL)z_fatal("desktopWindowPresentRGBA texture: %s",z_desktop_sdl.GetError());}if(!z_desktop_sdl.UpdateTexture((SDL_Texture*)w->frame_texture,NULL,pixels.as.buffer->data,(int)(width*4)))return z_bool(false);(void)z_desktop_sdl.RenderClear((SDL_Renderer*)w->renderer);if(!z_desktop_sdl.RenderTexture((SDL_Renderer*)w->renderer,(SDL_Texture*)w->frame_texture,NULL,NULL))return z_bool(false);return z_bool(z_desktop_sdl.RenderPresent((SDL_Renderer*)w->renderer));
#else
        return z_bool(false);
#endif
    }
    if(strcmp(name,"desktopWindowSetVSync")==0){z_expect_args(name,argc,2);ZDesktopWindowNative*w=z_desktop_expect_window(args[0]);w->vsync=z_as_bool(args[1]);
#if defined(ZUMBRA_DESKTOP_SDL3)
        if(!w->app->headless&&w->renderer!=NULL&&z_desktop_sdl.SetRenderVSync!=NULL&&!z_desktop_sdl.SetRenderVSync((SDL_Renderer*)w->renderer,w->vsync?1:0))z_fatal("desktopWindowSetVSync: %s",z_desktop_sdl.GetError());
#endif
        return args[0];
    }
    if(strcmp(name,"desktopEmit")==0){z_expect_args(name,argc,2);ZDesktopAppNative*app=z_desktop_expect_app(args[0]);ZValue event=args[1];if(event.tag!=ZV_DICT)z_fatal("desktopEmit event must be dictionary");const char*type=z_desktop_opt_string(event,"type","custom");int64_t window_id=z_desktop_opt_int(event,"windowId",0);ZValue data=z_desktop_dict_get(event,"data");if(data.tag!=ZV_DICT)data=z_dict_from(NULL,0);z_desktop_queue(app,type,window_id,data);return z_bool(true);}
    if(strcmp(name,"desktopSetClipboard")==0){z_expect_args(name,argc,2);ZDesktopAppNative*app=z_desktop_expect_app(args[0]);const char*text=z_as_cstring(args[1]);if(app->headless){app->clipboard=z_strdup(text);z_desktop_queue(app,"clipboard_updated",0,z_dict_from(NULL,0));return z_bool(true);}
#if defined(ZUMBRA_DESKTOP_SDL3)
        if (!z_desktop_sdl.SetClipboardText(text)) z_fatal("desktop clipboard: %s",z_desktop_sdl.GetError());
        return z_bool(true);
#else
        return z_bool(false);
#endif
    }
    if(strcmp(name,"desktopClipboard")==0){z_expect_args(name,argc,1);ZDesktopAppNative*app=z_desktop_expect_app(args[0]);if(app->headless)return z_string(app->clipboard==NULL?"":app->clipboard);
#if defined(ZUMBRA_DESKTOP_SDL3)
        char*text=z_desktop_sdl.GetClipboardText();if(text==NULL)z_fatal("desktop clipboard: %s",z_desktop_sdl.GetError());ZValue result=z_string(text);z_desktop_sdl.Free(text);return result;
#else
        return z_string("");
#endif
    }
    if(strcmp(name,"desktopPickFile")==0){z_expect_args(name,argc,2);return z_desktop_pick_file_native(args[1],false);}
    if(strcmp(name,"desktopPickFolder")==0){z_expect_args(name,argc,2);return z_desktop_pick_file_native(args[1],true);}
    if(strcmp(name,"desktopNotify")==0){z_expect_args(name,argc,2);ZDesktopAppNative*app=z_desktop_expect_app(args[0]);const char*title=z_desktop_opt_string(args[1],"title","Zumbra"),*body=z_desktop_opt_string(args[1],"body","");if(app->headless){z_desktop_queue(app,"notification",0,z_desktop_dict2("title",z_string(title),"body",z_string(body)));return z_bool(true);}return z_bool(z_desktop_notify_native(title,body));}
    if(strcmp(name,"desktopPaths")==0){z_expect_args(name,argc,1);(void)z_desktop_expect_app(args[0]);return z_desktop_paths_native();}
    if(strcmp(name,"desktopOpenExternal")==0){z_expect_args(name,argc,2);ZDesktopAppNative*app=z_desktop_expect_app(args[0]);const char*target=z_as_cstring(args[1]);if(app->headless){z_desktop_queue(app,"external_opened",0,z_desktop_dict1("target",z_string(target)));return z_bool(true);}if(!z_desktop_open_external_native(target))z_fatal("desktop external-resource backend is unavailable");return z_bool(true);}
    if(strcmp(name,"desktopTray")==0){z_expect_args(name,argc,2);ZDesktopAppNative*app=z_desktop_expect_app(args[0]);ZDesktopTrayNative*t=(ZDesktopTrayNative*)z_alloc(sizeof(*t));memset(t,0,sizeof(*t));t->app=app;t->open=true;
#if defined(ZUMBRA_DESKTOP_SDL3)
        if(!app->headless){if(z_desktop_sdl.CreateTray==NULL||z_desktop_sdl.CreateTrayMenu==NULL)z_fatal("SDL3 tray APIs are unavailable");const char*icon=z_desktop_opt_string(args[1],"icon","");SDL_Surface*surface=z_desktop_sdl_load_bmp(icon);SDL_Tray*tray=z_desktop_sdl.CreateTray(surface,z_desktop_opt_string(args[1],"tooltip","Zumbra"));if(surface!=NULL&&z_desktop_sdl.DestroySurface!=NULL)z_desktop_sdl.DestroySurface(surface);if(tray==NULL)z_fatal("could not create system tray");SDL_TrayMenu*menu=z_desktop_sdl.CreateTrayMenu(tray);if(menu==NULL){z_desktop_sdl.DestroyTray(tray);z_fatal("could not create tray menu");}t->handle=tray;t->menu=menu;}
#endif
        t->next=app->trays;app->trays=t;if(app->headless)z_desktop_queue(app,"tray_created",0,z_dict_from(NULL,0));return z_desktop_object(ZV_DESKTOP_TRAY,ZK_DESKTOP_TRAY,t);}
    if(strcmp(name,"desktopTrayAdd")==0){z_expect_args(name,argc,4);ZDesktopTrayNative*t=z_desktop_expect_tray(args[0]);const char*id=z_as_cstring(args[1]),*label=z_as_cstring(args[2]);if(!t->open)z_fatal("tray is closed");char key[320];snprintf(key,sizeof(key),"tray:%s",id);z_desktop_add_handler(&t->app->handlers,key,args[3]);
#if defined(ZUMBRA_DESKTOP_SDL3)
        if(!t->app->headless){SDL_TrayEntry*entry=z_desktop_sdl.InsertTrayEntryAt((SDL_TrayMenu*)t->menu,-1,label,1u);if(entry==NULL)z_fatal("could not create tray entry");ZDesktopTrayCallback*callback=(ZDesktopTrayCallback*)z_alloc(sizeof(*callback));callback->app=t->app;callback->id=z_strdup(id);z_desktop_sdl.SetTrayEntryCallback(entry,z_desktop_tray_callback,callback);}
#endif
        return args[0];}
    if(strcmp(name,"desktopTrayTooltip")==0){z_expect_args(name,argc,2);ZDesktopTrayNative*t=z_desktop_expect_tray(args[0]);if(!t->open)z_fatal("tray is closed");
#if defined(ZUMBRA_DESKTOP_SDL3)
        if(!t->app->headless&&z_desktop_sdl.SetTrayTooltip!=NULL)z_desktop_sdl.SetTrayTooltip((SDL_Tray*)t->handle,z_as_cstring(args[1]));
#endif
        return args[0];}
    if(strcmp(name,"desktopTrayClose")==0||strcmp(name,"desktopTrayOpen")==0){z_expect_args(name,argc,1);ZDesktopTrayNative*t=z_desktop_expect_tray(args[0]);if(strcmp(name,"desktopTrayOpen")==0)return z_bool(t->open);if(t->open){
#if defined(ZUMBRA_DESKTOP_SDL3)
            if(!t->app->headless&&t->handle!=NULL&&z_desktop_sdl.DestroyTray!=NULL)z_desktop_sdl.DestroyTray((SDL_Tray*)t->handle);
#endif
            t->open=false;}return z_bool(true);}
    if(strcmp(name,"desktopSpawn")==0){z_expect_args(name,argc,2);return z_desktop_process_spawn(z_as_cstring(args[0]),args[1]);}
    if(strcmp(name,"desktopProcessWait")==0){z_expect_args(name,argc,1);ZDesktopProcessNative*p=z_desktop_expect_process(args[0]);if(!p->waited){
#if defined(_WIN32)
        if(WaitForSingleObject(p->handle,INFINITE)!=WAIT_OBJECT_0)z_fatal("could not wait for process");DWORD code=0;if(!GetExitCodeProcess(p->handle,&code))z_fatal("could not read process exit code");p->exit_code=(int64_t)code;CloseHandle(p->handle);p->handle=NULL;
#else
        int status=0;if(waitpid(p->pid,&status,0)<0)z_fatal("waitpid failed: %s",strerror(errno));p->exit_code=WIFEXITED(status)?WEXITSTATUS(status):(WIFSIGNALED(status)?128+WTERMSIG(status):-1);
#endif
        p->waited=true;p->running=false;}return z_int(p->exit_code);}
    if(strcmp(name,"desktopProcessKill")==0){z_expect_args(name,argc,1);ZDesktopProcessNative*p=z_desktop_expect_process(args[0]);if(p->running){
#if defined(_WIN32)
        if(!TerminateProcess(p->handle,1))z_fatal("could not terminate process: Windows error %lu",(unsigned long)GetLastError());
#else
        if(kill(p->pid,SIGTERM)!=0&&errno!=ESRCH)z_fatal("could not kill process: %s",strerror(errno));
#endif
    }return z_bool(true);}
    if(strcmp(name,"desktopProcessRunning")==0){z_expect_args(name,argc,1);ZDesktopProcessNative*p=z_desktop_expect_process(args[0]);if(p->running){
#if defined(_WIN32)
        DWORD code=STILL_ACTIVE;if(!GetExitCodeProcess(p->handle,&code))z_fatal("could not read process state");if(code!=STILL_ACTIVE){p->running=false;p->waited=true;p->exit_code=(int64_t)code;CloseHandle(p->handle);p->handle=NULL;}
#else
        int status=0;pid_t state=waitpid(p->pid,&status,WNOHANG);if(state==p->pid){p->running=false;p->waited=true;p->exit_code=WIFEXITED(status)?WEXITSTATUS(status):-1;}
#endif
    }return z_bool(p->running);}
    if(strcmp(name,"desktopProcessId")==0){z_expect_args(name,argc,1);return z_int((int64_t)z_desktop_expect_process(args[0])->pid);}
    if(strncmp(name,"desktopWindow",13)==0){ZDesktopWindowNative*w=z_desktop_expect_window(args[0]);if(strcmp(name,"desktopWindowShow")==0){z_expect_args(name,argc,1);return z_desktop_window_action(w,"show");}if(strcmp(name,"desktopWindowHide")==0){z_expect_args(name,argc,1);return z_desktop_window_action(w,"hide");}if(strcmp(name,"desktopWindowClose")==0){z_expect_args(name,argc,1);z_desktop_close_window_native(w);return args[0];}if(strcmp(name,"desktopWindowOpen")==0){z_expect_args(name,argc,1);return z_bool(w->open);}if(strcmp(name,"desktopWindowId")==0){z_expect_args(name,argc,1);return z_int(w->id);}if(strcmp(name,"desktopWindowTitle")==0){z_expect_args(name,argc,1);
#if defined(ZUMBRA_DESKTOP_SDL3)
            if(!w->app->headless&&w->open)w->title=z_strdup(z_desktop_sdl.GetWindowTitle((SDL_Window*)w->handle));
#endif
            return z_string(w->title);}if(strcmp(name,"desktopWindowSetTitle")==0){z_expect_args(name,argc,2);const char*title=z_as_cstring(args[1]);w->title=z_strdup(title);
#if defined(ZUMBRA_DESKTOP_SDL3)
            if(!w->app->headless&&!z_desktop_sdl.SetWindowTitle((SDL_Window*)w->handle,title))z_fatal("could not set window title");
#endif
            return args[0];}if(strcmp(name,"desktopWindowSize")==0||strcmp(name,"desktopWindowPixelSize")==0){z_expect_args(name,argc,1);int64_t a=w->width,b=w->height;
#if defined(ZUMBRA_DESKTOP_SDL3)
            if(!w->app->headless){int x=0,y=0;bool ok=strcmp(name,"desktopWindowPixelSize")==0?z_desktop_sdl.GetWindowSizeInPixels((SDL_Window*)w->handle,&x,&y):z_desktop_sdl.GetWindowSize((SDL_Window*)w->handle,&x,&y);if(ok){a=x;b=y;}}
#endif
            return z_desktop_dict2("width",z_int(a),"height",z_int(b));}if(strcmp(name,"desktopWindowSetSize")==0){z_expect_args(name,argc,3);int64_t width=z_as_i64(args[1]),height=z_as_i64(args[2]);if(width<1||height<1)z_fatal("window width and height must be positive");w->width=width;w->height=height;w->pixel_width=(int64_t)(width*w->density);w->pixel_height=(int64_t)(height*w->density);
#if defined(ZUMBRA_DESKTOP_SDL3)
            if(!w->app->headless&&!z_desktop_sdl.SetWindowSize((SDL_Window*)w->handle,(int)width,(int)height))z_fatal("could not resize window");
#endif
            if (w->app->headless) z_desktop_queue(w->app,"window_resized",w->id,z_desktop_dict2("width",z_int(width),"height",z_int(height)));
            return args[0];
        }
        if(strcmp(name,"desktopWindowPosition")==0){z_expect_args(name,argc,1);int64_t x=w->x,y=w->y;
#if defined(ZUMBRA_DESKTOP_SDL3)
            if(!w->app->headless){int a=0,b=0;if(z_desktop_sdl.GetWindowPosition((SDL_Window*)w->handle,&a,&b)){x=a;y=b;}}
#endif
            return z_desktop_dict2("x",z_int(x),"y",z_int(y));}if(strcmp(name,"desktopWindowSetPosition")==0){z_expect_args(name,argc,3);w->x=z_as_i64(args[1]);w->y=z_as_i64(args[2]);
#if defined(ZUMBRA_DESKTOP_SDL3)
            if(!w->app->headless&&!z_desktop_sdl.SetWindowPosition((SDL_Window*)w->handle,(int)w->x,(int)w->y))z_fatal("could not move window");
#endif
            if (w->app->headless) z_desktop_queue(w->app,"window_moved",w->id,z_desktop_dict2("x",z_int(w->x),"y",z_int(w->y)));
            return args[0];
        }
        if(strcmp(name,"desktopWindowFullscreen")==0){z_expect_args(name,argc,1);
#if defined(ZUMBRA_DESKTOP_SDL3)
            if(!w->app->headless)w->fullscreen=(z_desktop_sdl.GetWindowFlags((SDL_Window*)w->handle)&0x1u)!=0;
#endif
            return z_bool(w->fullscreen);}if(strcmp(name,"desktopWindowSetFullscreen")==0){z_expect_args(name,argc,2);w->fullscreen=z_as_bool(args[1]);
#if defined(ZUMBRA_DESKTOP_SDL3)
            if(!w->app->headless&&!z_desktop_sdl.SetWindowFullscreen((SDL_Window*)w->handle,w->fullscreen))z_fatal("could not change fullscreen");
#endif
            if (w->app->headless) z_desktop_queue(w->app,w->fullscreen?"window_enter_fullscreen":"window_leave_fullscreen",w->id,z_dict_from(NULL,0));
            return args[0];
        }
        if(strcmp(name,"desktopWindowMaximize")==0){z_expect_args(name,argc,1);return z_desktop_window_action(w,"maximize");}if(strcmp(name,"desktopWindowMinimize")==0){z_expect_args(name,argc,1);return z_desktop_window_action(w,"minimize");}if(strcmp(name,"desktopWindowRestore")==0){z_expect_args(name,argc,1);return z_desktop_window_action(w,"restore");}if(strcmp(name,"desktopWindowFocus")==0){z_expect_args(name,argc,1);return z_desktop_window_action(w,"focus");}if(strcmp(name,"desktopWindowDisplayScale")==0){z_expect_args(name,argc,1);
#if defined(ZUMBRA_DESKTOP_SDL3)
            if(!w->app->headless)w->scale=z_desktop_sdl.GetWindowDisplayScale((SDL_Window*)w->handle);
#endif
            return z_float(w->scale);}if(strcmp(name,"desktopWindowPixelDensity")==0){z_expect_args(name,argc,1);
#if defined(ZUMBRA_DESKTOP_SDL3)
            if(!w->app->headless)w->density=z_desktop_sdl.GetWindowPixelDensity((SDL_Window*)w->handle);
#endif
            return z_float(w->density);}if(strcmp(name,"desktopWindowSetIcon")==0){z_expect_args(name,argc,2);
#if defined(ZUMBRA_DESKTOP_SDL3)
            if(!w->app->headless){if(z_desktop_sdl.IOFromFile==NULL||z_desktop_sdl.LoadBMP_IO==NULL||z_desktop_sdl.SetWindowIcon==NULL)z_fatal("SDL3 BMP icon APIs are unavailable");SDL_Surface*surface=z_desktop_sdl_load_bmp(z_as_cstring(args[1]));if(surface==NULL)z_fatal("could not load BMP icon");bool ok=z_desktop_sdl.SetWindowIcon((SDL_Window*)w->handle,surface);z_desktop_sdl.DestroySurface(surface);if(!ok)z_fatal("could not set window icon");}
#endif
            return args[0];}}
    if (handled != NULL) *handled = false;
    return z_null();
}

static void z_desktop_runtime_init(void) {
#if defined(ZUMBRA_DESKTOP_SDL3)
    memset(&z_desktop_sdl,0,sizeof(z_desktop_sdl));
#endif
}
static void z_desktop_runtime_shutdown(void) {
#if defined(ZUMBRA_DESKTOP_SDL3)
    if(z_desktop_sdl.initialized&&z_desktop_sdl.Quit!=NULL){z_desktop_destroy_system_cursors();z_desktop_sdl.Quit();}
    if(z_desktop_sdl.lib!=NULL)z_desktop_library_close(z_desktop_sdl.lib);
    memset(&z_desktop_sdl,0,sizeof(z_desktop_sdl));
#endif
}

#endif
