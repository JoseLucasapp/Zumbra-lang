#if defined(ZUMBRA_ENABLE_DESKTOP)

/* Z13 Linux desktop runtime. SDL3 is loaded dynamically so a generated
 * executable can still run in deterministic headless mode on CI servers. */
#include <ctype.h>
#include <dirent.h>
#include <fcntl.h>
#include <sys/wait.h>
#include <sys/types.h>
#include <unistd.h>
#ifndef PATH_MAX
#define PATH_MAX 4096
#endif
#if defined(__linux__)
#include <dlfcn.h>
#endif

typedef struct ZDesktopAppNative ZDesktopAppNative;
typedef struct ZDesktopWindowNative ZDesktopWindowNative;
typedef struct ZDesktopTrayNative ZDesktopTrayNative;
typedef struct ZDesktopProcessNative ZDesktopProcessNative;

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
    pid_t pid;
    bool running;
    bool waited;
    int64_t exit_code;
};

struct ZDesktopAppNative {
    pthread_mutex_t mutex;
    bool headless, running, closed;
    bool close_on_request, quit_on_last_window;
    int64_t poll_interval_ms;
    int64_t next_window_id;
    char *clipboard;
    ZDesktopWindowNative *windows;
    ZDesktopTrayNative *trays;
    ZDesktopHandlerNative *handlers;
    ZDesktopHandlerNative *shortcuts;
    ZDesktopEventNative *event_head, *event_tail;
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
    struct timespec ts; clock_gettime(CLOCK_REALTIME, &ts);
    return (int64_t)ts.tv_sec * INT64_C(1000000000) + (int64_t)ts.tv_nsec;
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
#if defined(__linux__)
typedef struct SDL_Window SDL_Window;
typedef struct SDL_Surface SDL_Surface;
typedef struct SDL_IOStream SDL_IOStream;
typedef struct SDL_Tray SDL_Tray;
typedef struct SDL_TrayMenu SDL_TrayMenu;
typedef struct SDL_TrayEntry SDL_TrayEntry;
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
typedef bool (*ZD_SDL_PollEvent)(ZDesktopSDLEvent*); typedef bool (*ZD_SDL_WaitEvent)(ZDesktopSDLEvent*); typedef bool (*ZD_SDL_WaitEventTimeout)(ZDesktopSDLEvent*,int32_t); typedef const char *(*ZD_SDL_GetKeyName)(uint32_t);
typedef bool (*ZD_SDL_SetClipboardText)(const char*); typedef char *(*ZD_SDL_GetClipboardText)(void); typedef void (*ZD_SDL_free)(void*);
typedef SDL_IOStream *(*ZD_SDL_IOFromFile)(const char*,const char*); typedef SDL_Surface *(*ZD_SDL_LoadBMP_IO)(SDL_IOStream*,bool); typedef void (*ZD_SDL_DestroySurface)(SDL_Surface*); typedef bool (*ZD_SDL_SetWindowIcon)(SDL_Window*,SDL_Surface*);
typedef SDL_Tray *(*ZD_SDL_CreateTray)(SDL_Surface*,const char*); typedef SDL_TrayMenu *(*ZD_SDL_CreateTrayMenu)(SDL_Tray*); typedef SDL_TrayEntry *(*ZD_SDL_InsertTrayEntryAt)(SDL_TrayMenu*,int,const char*,uint32_t); typedef void (*ZD_SDL_SetTrayEntryCallback)(SDL_TrayEntry*,void(*)(void*,SDL_TrayEntry*),void*); typedef void (*ZD_SDL_SetTrayTooltip)(SDL_Tray*,const char*); typedef void (*ZD_SDL_DestroyTray)(SDL_Tray*);

typedef struct {
    void *lib; bool initialized; char error[512];
    ZD_SDL_Init Init; ZD_SDL_Quit Quit; ZD_SDL_GetError GetError; ZD_SDL_SetAppMetadata SetAppMetadata;
    ZD_SDL_CreateWindow CreateWindow; ZD_SDL_DestroyWindow DestroyWindow; ZD_SDL_GetWindowID GetWindowID; ZD_SDL_ShowWindow ShowWindow; ZD_SDL_HideWindow HideWindow; ZD_SDL_GetWindowTitle GetWindowTitle; ZD_SDL_SetWindowTitle SetWindowTitle;
    ZD_SDL_GetWindowSize GetWindowSize; ZD_SDL_GetWindowSizeInPixels GetWindowSizeInPixels; ZD_SDL_SetWindowSize SetWindowSize; ZD_SDL_GetWindowPosition GetWindowPosition; ZD_SDL_SetWindowPosition SetWindowPosition; ZD_SDL_SetWindowFullscreen SetWindowFullscreen; ZD_SDL_GetWindowFlags GetWindowFlags;
    ZD_SDL_MaximizeWindow MaximizeWindow; ZD_SDL_MinimizeWindow MinimizeWindow; ZD_SDL_RestoreWindow RestoreWindow; ZD_SDL_RaiseWindow RaiseWindow; ZD_SDL_GetWindowDisplayScale GetWindowDisplayScale; ZD_SDL_GetWindowPixelDensity GetWindowPixelDensity;
    ZD_SDL_PollEvent PollEvent; ZD_SDL_WaitEvent WaitEvent; ZD_SDL_WaitEventTimeout WaitEventTimeout; ZD_SDL_GetKeyName GetKeyName; ZD_SDL_SetClipboardText SetClipboardText; ZD_SDL_GetClipboardText GetClipboardText; ZD_SDL_free Free;
    ZD_SDL_IOFromFile IOFromFile; ZD_SDL_LoadBMP_IO LoadBMP_IO; ZD_SDL_DestroySurface DestroySurface; ZD_SDL_SetWindowIcon SetWindowIcon;
    ZD_SDL_CreateTray CreateTray; ZD_SDL_CreateTrayMenu CreateTrayMenu; ZD_SDL_InsertTrayEntryAt InsertTrayEntryAt; ZD_SDL_SetTrayEntryCallback SetTrayEntryCallback; ZD_SDL_SetTrayTooltip SetTrayTooltip; ZD_SDL_DestroyTray DestroyTray;
} ZDesktopSDL;
static ZDesktopSDL z_desktop_sdl;
#define ZD_LOAD_REQUIRED(field, symbol, type) do { z_desktop_sdl.field=(type)dlsym(z_desktop_sdl.lib,symbol); if(z_desktop_sdl.field==NULL){snprintf(z_desktop_sdl.error,sizeof(z_desktop_sdl.error),"missing SDL3 symbol %s",symbol);return false;} } while(0)
#define ZD_LOAD_OPTIONAL(field, symbol, type) do { z_desktop_sdl.field=(type)dlsym(z_desktop_sdl.lib,symbol); } while(0)
static bool z_desktop_sdl_load(void) {
    if (z_desktop_sdl.initialized) return true;
    memset(&z_desktop_sdl, 0, sizeof(z_desktop_sdl));
    const char *names[] = {"libSDL3.so.0", "libSDL3.so", NULL};
    for (size_t i=0; names[i] != NULL; i++) { z_desktop_sdl.lib=dlopen(names[i],RTLD_NOW|RTLD_LOCAL); if(z_desktop_sdl.lib!=NULL)break; }
    if (z_desktop_sdl.lib == NULL) { snprintf(z_desktop_sdl.error,sizeof(z_desktop_sdl.error),"SDL3 shared library was not found"); return false; }
    ZD_LOAD_REQUIRED(Init,"SDL_Init",ZD_SDL_Init); ZD_LOAD_REQUIRED(Quit,"SDL_Quit",ZD_SDL_Quit); ZD_LOAD_REQUIRED(GetError,"SDL_GetError",ZD_SDL_GetError);
    ZD_LOAD_OPTIONAL(SetAppMetadata,"SDL_SetAppMetadata",ZD_SDL_SetAppMetadata);
    ZD_LOAD_REQUIRED(CreateWindow,"SDL_CreateWindow",ZD_SDL_CreateWindow); ZD_LOAD_REQUIRED(DestroyWindow,"SDL_DestroyWindow",ZD_SDL_DestroyWindow); ZD_LOAD_REQUIRED(GetWindowID,"SDL_GetWindowID",ZD_SDL_GetWindowID);
    ZD_LOAD_REQUIRED(ShowWindow,"SDL_ShowWindow",ZD_SDL_ShowWindow); ZD_LOAD_REQUIRED(HideWindow,"SDL_HideWindow",ZD_SDL_HideWindow); ZD_LOAD_REQUIRED(GetWindowTitle,"SDL_GetWindowTitle",ZD_SDL_GetWindowTitle); ZD_LOAD_REQUIRED(SetWindowTitle,"SDL_SetWindowTitle",ZD_SDL_SetWindowTitle);
    ZD_LOAD_REQUIRED(GetWindowSize,"SDL_GetWindowSize",ZD_SDL_GetWindowSize); ZD_LOAD_REQUIRED(GetWindowSizeInPixels,"SDL_GetWindowSizeInPixels",ZD_SDL_GetWindowSizeInPixels); ZD_LOAD_REQUIRED(SetWindowSize,"SDL_SetWindowSize",ZD_SDL_SetWindowSize); ZD_LOAD_REQUIRED(GetWindowPosition,"SDL_GetWindowPosition",ZD_SDL_GetWindowPosition); ZD_LOAD_REQUIRED(SetWindowPosition,"SDL_SetWindowPosition",ZD_SDL_SetWindowPosition); ZD_LOAD_REQUIRED(SetWindowFullscreen,"SDL_SetWindowFullscreen",ZD_SDL_SetWindowFullscreen); ZD_LOAD_REQUIRED(GetWindowFlags,"SDL_GetWindowFlags",ZD_SDL_GetWindowFlags);
    ZD_LOAD_REQUIRED(MaximizeWindow,"SDL_MaximizeWindow",ZD_SDL_MaximizeWindow); ZD_LOAD_REQUIRED(MinimizeWindow,"SDL_MinimizeWindow",ZD_SDL_MinimizeWindow); ZD_LOAD_REQUIRED(RestoreWindow,"SDL_RestoreWindow",ZD_SDL_RestoreWindow); ZD_LOAD_REQUIRED(RaiseWindow,"SDL_RaiseWindow",ZD_SDL_RaiseWindow); ZD_LOAD_REQUIRED(GetWindowDisplayScale,"SDL_GetWindowDisplayScale",ZD_SDL_GetWindowDisplayScale); ZD_LOAD_REQUIRED(GetWindowPixelDensity,"SDL_GetWindowPixelDensity",ZD_SDL_GetWindowPixelDensity);
    ZD_LOAD_REQUIRED(PollEvent,"SDL_PollEvent",ZD_SDL_PollEvent); ZD_LOAD_REQUIRED(WaitEvent,"SDL_WaitEvent",ZD_SDL_WaitEvent); ZD_LOAD_REQUIRED(WaitEventTimeout,"SDL_WaitEventTimeout",ZD_SDL_WaitEventTimeout); ZD_LOAD_REQUIRED(GetKeyName,"SDL_GetKeyName",ZD_SDL_GetKeyName); ZD_LOAD_REQUIRED(SetClipboardText,"SDL_SetClipboardText",ZD_SDL_SetClipboardText); ZD_LOAD_REQUIRED(GetClipboardText,"SDL_GetClipboardText",ZD_SDL_GetClipboardText); ZD_LOAD_REQUIRED(Free,"SDL_free",ZD_SDL_free);
    ZD_LOAD_OPTIONAL(IOFromFile,"SDL_IOFromFile",ZD_SDL_IOFromFile); ZD_LOAD_OPTIONAL(LoadBMP_IO,"SDL_LoadBMP_IO",ZD_SDL_LoadBMP_IO); ZD_LOAD_OPTIONAL(DestroySurface,"SDL_DestroySurface",ZD_SDL_DestroySurface); ZD_LOAD_OPTIONAL(SetWindowIcon,"SDL_SetWindowIcon",ZD_SDL_SetWindowIcon);
    ZD_LOAD_OPTIONAL(CreateTray,"SDL_CreateTray",ZD_SDL_CreateTray); ZD_LOAD_OPTIONAL(CreateTrayMenu,"SDL_CreateTrayMenu",ZD_SDL_CreateTrayMenu); ZD_LOAD_OPTIONAL(InsertTrayEntryAt,"SDL_InsertTrayEntryAt",ZD_SDL_InsertTrayEntryAt); ZD_LOAD_OPTIONAL(SetTrayEntryCallback,"SDL_SetTrayEntryCallback",ZD_SDL_SetTrayEntryCallback); ZD_LOAD_OPTIONAL(SetTrayTooltip,"SDL_SetTrayTooltip",ZD_SDL_SetTrayTooltip); ZD_LOAD_OPTIONAL(DestroyTray,"SDL_DestroyTray",ZD_SDL_DestroyTray);
    if (!z_desktop_sdl.Init(0x00000020u)) { snprintf(z_desktop_sdl.error,sizeof(z_desktop_sdl.error),"%s",z_desktop_sdl.GetError()); return false; }
    z_desktop_sdl.initialized=true; return true;
}
static SDL_Surface *z_desktop_sdl_load_bmp(const char *path) {
    if(path==NULL||*path=='\0'||z_desktop_sdl.IOFromFile==NULL||z_desktop_sdl.LoadBMP_IO==NULL)return NULL;
    SDL_IOStream *stream=z_desktop_sdl.IOFromFile(path,"rb"); if(stream==NULL)return NULL; return z_desktop_sdl.LoadBMP_IO(stream,true);
}
#else
static bool z_desktop_sdl_load(void) { return false; }
#endif

/* -------- event queue and handlers -------- */
static void z_desktop_queue(ZDesktopAppNative *app, const char *type, int64_t window_id, ZValue data) {
    ZDesktopEventNative *event=(ZDesktopEventNative *)z_alloc(sizeof(*event));
    event->type=z_strdup(type); event->window_id=window_id; event->timestamp=z_desktop_now_ns(); event->data=data; event->next=NULL;
    pthread_mutex_lock(&app->mutex);
    if(app->event_tail!=NULL)app->event_tail->next=event;else app->event_head=event;
    app->event_tail=event;
    pthread_mutex_unlock(&app->mutex);
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
#if defined(__linux__)
    if(!window->app->headless&&window->handle!=NULL){z_desktop_sdl.DestroyWindow((SDL_Window *)window->handle);window->handle=NULL;}
#endif
    z_desktop_queue(window->app,"window_closed",window->id,z_dict_from(NULL,0));
}
static void z_desktop_dispatch(ZDesktopAppNative *app, ZDesktopEventNative *event) {
    if (event == NULL) return;
    ZValue event_value = z_desktop_event_value(event);
    for(ZDesktopHandlerNative *h=app->handlers;h!=NULL;h=h->next){if(strcmp(h->name,event->type)==0||strcmp(h->name,"event")==0){(void)z_call(h->handler,&event_value,1);}}
    if(strcmp(event->type,"key_down")==0){ZValue raw=z_desktop_dict_get(event->data,"shortcut");if(raw.tag==ZV_STRING){char normalized[256];z_desktop_normalize_shortcut(raw.as.s,normalized,sizeof(normalized));for(ZDesktopHandlerNative*h=app->shortcuts;h!=NULL;h=h->next)if(strcmp(h->name,normalized)==0)(void)z_call(h->handler,&event_value,1);}}
    if(strcmp(event->type,"tray")==0){ZValue id=z_desktop_dict_get(event->data,"id");if(id.tag==ZV_STRING){char key[320];snprintf(key,sizeof(key),"tray:%s",id.as.s);for(ZDesktopHandlerNative*h=app->handlers;h!=NULL;h=h->next)if(strcmp(h->name,key)==0)(void)z_call(h->handler,&event_value,1);}}
    if(strcmp(event->type,"window_close_requested")==0&&app->close_on_request)z_desktop_close_window_native(z_desktop_find_window(app,event->window_id));
    if((strcmp(event->type,"window_closed")==0)&&app->quit_on_last_window&&z_desktop_open_windows(app)==0)app->running=false;
    if(strcmp(event->type,"quit")==0||strcmp(event->type,"terminating")==0)app->running=false;
}

#if defined(__linux__)
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
    if(common->type==0x100u)type="quit"; else if(common->type==0x101u)type="terminating"; else if(common->type==0x102u)type="low_memory"; else if(common->type==0x103u)type="will_enter_background"; else if(common->type==0x104u)type="did_enter_background"; else if(common->type==0x105u)type="will_enter_foreground"; else if(common->type==0x106u)type="did_enter_foreground"; else if(common->type==0x109u)type="system_theme_changed";
    else if(common->type>=0x202u&&common->type<=0x220u){ZDesktopSDLWindowEvent*w=(ZDesktopSDLWindowEvent*)&raw;window_id=w->windowID;switch(common->type){case 0x202:type="window_shown";break;case 0x203:type="window_hidden";break;case 0x204:type="window_exposed";break;case 0x205:type="window_moved";data=z_desktop_dict2("x",z_int(w->data1),"y",z_int(w->data2));break;case 0x206:type="window_resized";data=z_desktop_dict2("width",z_int(w->data1),"height",z_int(w->data2));break;case 0x207:type="window_pixel_size_changed";data=z_desktop_dict2("width",z_int(w->data1),"height",z_int(w->data2));break;case 0x209:type="window_minimized";break;case 0x20a:type="window_maximized";break;case 0x20b:type="window_restored";break;case 0x20c:type="window_mouse_enter";break;case 0x20d:type="window_mouse_leave";break;case 0x20e:type="window_focus_gained";break;case 0x20f:type="window_focus_lost";break;case 0x210:type="window_close_requested";break;case 0x219:type="window_enter_fullscreen";break;case 0x21a:type="window_leave_fullscreen";break;case 0x21b:type="window_closed";break;default:type="window_event";break;}}
    else if(common->type==0x300u||common->type==0x301u){ZDesktopSDLKeyboardEvent*k=(ZDesktopSDLKeyboardEvent*)&raw;window_id=k->windowID;char shortcut[256];z_desktop_shortcut(k->key,k->mod,shortcut,sizeof(shortcut));const char*key=z_desktop_sdl.GetKeyName(k->key);ZValue pairs[5]={z_pair(z_string("key"),z_string(key==NULL?"":key)),z_pair(z_string("keycode"),z_int(k->key)),z_pair(z_string("modifiers"),z_int(k->mod)),z_pair(z_string("repeat"),z_bool(k->repeat)),z_pair(z_string("shortcut"),z_string(shortcut))};data=z_dict_from(pairs,5);type=common->type==0x300u?"key_down":"key_up";}
    else if(common->type==0x303u){ZDesktopSDLTextEvent*t=(ZDesktopSDLTextEvent*)&raw;window_id=t->windowID;data=z_desktop_dict1("text",z_string(t->text==NULL?"":t->text));type="text_input";}
    else if(common->type==0x400u){ZDesktopSDLMouseMotionEvent*m=(ZDesktopSDLMouseMotionEvent*)&raw;window_id=m->windowID;ZValue pairs[4]={z_pair(z_string("x"),z_float(m->x)),z_pair(z_string("y"),z_float(m->y)),z_pair(z_string("dx"),z_float(m->xrel)),z_pair(z_string("dy"),z_float(m->yrel))};data=z_dict_from(pairs,4);type="mouse_move";}
    else if(common->type==0x401u||common->type==0x402u){ZDesktopSDLMouseButtonEvent*b=(ZDesktopSDLMouseButtonEvent*)&raw;window_id=b->windowID;ZValue pairs[4]={z_pair(z_string("button"),z_int(b->button)),z_pair(z_string("clicks"),z_int(b->clicks)),z_pair(z_string("x"),z_float(b->x)),z_pair(z_string("y"),z_float(b->y))};data=z_dict_from(pairs,4);type=common->type==0x401u?"mouse_down":"mouse_up";}
    else if(common->type==0x403u){ZDesktopSDLMouseWheelEvent*w=(ZDesktopSDLMouseWheelEvent*)&raw;window_id=w->windowID;data=z_desktop_dict2("x",z_float(w->x),"y",z_float(w->y));type="mouse_wheel";}
    else if(common->type==0x900u)type="clipboard_updated";
    else if(common->type>=0x1000u&&common->type<=0x1004u){ZDesktopSDLDropEvent*d=(ZDesktopSDLDropEvent*)&raw;window_id=d->windowID;if(common->type==0x1000u){ZValue pairs[4]={z_pair(z_string("path"),z_string(d->data==NULL?"":d->data)),z_pair(z_string("source"),z_string(d->source==NULL?"":d->source)),z_pair(z_string("x"),z_float(d->x)),z_pair(z_string("y"),z_float(d->y))};data=z_dict_from(pairs,4);type="drop_file";}else if(common->type==0x1001u){data=z_desktop_dict1("text",z_string(d->data==NULL?"":d->data));type="drop_text";}else if(common->type==0x1002u)type="drop_begin";else if(common->type==0x1003u)type="drop_complete";}
    ZDesktopEventNative *event=(ZDesktopEventNative*)z_alloc(sizeof(*event));event->type=z_strdup(type);event->window_id=window_id;event->timestamp=(int64_t)common->timestamp;event->data=data;event->next=NULL;return event;
}
#endif
static ZDesktopEventNative *z_desktop_next_event(ZDesktopAppNative *app,int64_t timeout_ms){
    ZDesktopEventNative*queued=z_desktop_dequeue(app);if(queued!=NULL)return queued;
#if defined(__linux__)
    if(!app->headless)return z_desktop_sdl_event(timeout_ms);
#endif
    if(timeout_ms>0){struct timespec request={.tv_sec=(time_t)(timeout_ms/1000),.tv_nsec=(long)((timeout_ms%1000)*1000000)};nanosleep(&request,NULL);}return z_desktop_dequeue(app);
}

/* -------- integration helpers -------- */
static char *z_desktop_shell_quote(const char *value){size_t n=3;for(const char*p=value;*p;p++)n+=(*p=='\''?4:1);char*out=(char*)malloc(n);if(out==NULL)z_fatal("out of memory");size_t i=0;out[i++]='\'';for(const char*p=value;*p;p++){if(*p=='\''){memcpy(out+i,"'\\''",4);i+=4;}else out[i++]=*p;}out[i++]='\'';out[i]='\0';return out;}
static char *z_desktop_capture(const char *command){FILE*pipe=popen(command,"r");if(pipe==NULL)return NULL;char*result=(char*)malloc(8192);if(result==NULL){pclose(pipe);return NULL;}size_t n=fread(result,1,8191,pipe);int status=pclose(pipe);if(status!=0){free(result);return NULL;}while(n>0&&(result[n-1]=='\n'||result[n-1]=='\r'))n--;result[n]='\0';return result;}
static bool z_desktop_command_exists(const char *name){char command[512];snprintf(command,sizeof(command),"command -v %s >/dev/null 2>&1",name);return system(command)==0;}
static void z_desktop_spawn_detached(const char *command,const char *a,const char*b){pid_t pid=fork();if(pid<0)return;if(pid==0){pid_t child=fork();if(child<0)_exit(127);if(child>0)_exit(0);setsid();if(b!=NULL)execlp(command,command,a,b,(char*)NULL);else if(a!=NULL)execlp(command,command,a,(char*)NULL);else execlp(command,command,(char*)NULL);_exit(127);}int status=0;while(waitpid(pid,&status,0)<0&&errno==EINTR){}}
static ZValue z_desktop_pick_file_native(ZValue options,bool folder){
    const char*default_path=z_desktop_opt_string(options,"defaultPath","");const char*title=z_desktop_opt_string(options,"title","");bool multiple=z_desktop_opt_bool(options,"multiple",false);
    char command[4096];char*qp=z_desktop_shell_quote(default_path);char*qt=z_desktop_shell_quote(title);
    if(z_desktop_command_exists("zenity")){snprintf(command,sizeof(command),"zenity --file-selection %s %s",folder?"--directory":"",multiple&&!folder?"--multiple --separator='\\n'":"");if(*title){size_t used=strlen(command);snprintf(command+used,sizeof(command)-used," --title=%s",qt);}if(*default_path){size_t used=strlen(command);snprintf(command+used,sizeof(command)-used," --filename=%s",qp);}}
    else if(z_desktop_command_exists("kdialog")){snprintf(command,sizeof(command),"kdialog %s %s",folder?"--getexistingdirectory":"--getopenfilename",qp);}else{free(qp);free(qt);return folder?z_null():z_array_from(NULL,0);}
    char*out=z_desktop_capture(command);free(qp);free(qt);if(out==NULL||*out=='\0'){free(out);return folder?z_null():z_array_from(NULL,0);}if(folder){ZValue result=z_string(out);free(out);return result;}
    ZValue items[128];size_t count=0;char*save=NULL;for(char*line=strtok_r(out,"\n",&save);line!=NULL&&count<128;line=strtok_r(NULL,"\n",&save))items[count++]=z_string(line);free(out);return z_array_from(items,count);
}
static ZValue z_desktop_paths_native(void){char home[PATH_MAX]="",cwd[PATH_MAX]="",exe[PATH_MAX]="";const char*h=getenv("HOME");if(h!=NULL)snprintf(home,sizeof(home),"%s",h);if(getcwd(cwd,sizeof(cwd))==NULL)cwd[0]='\0';ssize_t n=readlink("/proc/self/exe",exe,sizeof(exe)-1);if(n>=0)exe[n]='\0';else exe[0]='\0';const char*cache=getenv("XDG_CACHE_HOME");const char*config=getenv("XDG_CONFIG_HOME");const char*data=getenv("XDG_DATA_HOME");const char*runtime_dir=getenv("XDG_RUNTIME_DIR");char cache_buf[PATH_MAX],config_buf[PATH_MAX],data_buf[PATH_MAX];if(cache==NULL){snprintf(cache_buf,sizeof(cache_buf),"%s/.cache",home);cache=cache_buf;}if(config==NULL){snprintf(config_buf,sizeof(config_buf),"%s/.config",home);config=config_buf;}if(data==NULL){snprintf(data_buf,sizeof(data_buf),"%s/.local/share",home);data=data_buf;}ZValue pairs[8]={z_pair(z_string("home"),z_string(home)),z_pair(z_string("cache"),z_string(cache)),z_pair(z_string("config"),z_string(config)),z_pair(z_string("data"),z_string(data)),z_pair(z_string("runtime"),z_string(runtime_dir==NULL?"":runtime_dir)),z_pair(z_string("executable"),z_string(exe)),z_pair(z_string("cwd"),z_string(cwd)),z_pair(z_string("temp"),z_string(getenv("TMPDIR")?getenv("TMPDIR"):"/tmp"))};return z_dict_from(pairs,8);}

/* -------- app and window primitives -------- */
static ZValue z_desktop_app_new(ZValue options){
    if (options.tag != ZV_DICT) z_fatal("desktopApp expects options dictionary");
    ZDesktopAppNative*app=(ZDesktopAppNative*)z_alloc(sizeof(*app));memset(app,0,sizeof(*app));pthread_mutex_init(&app->mutex,NULL);app->close_on_request=z_desktop_opt_bool(options,"closeOnRequest",true);app->quit_on_last_window=z_desktop_opt_bool(options,"quitOnLastWindow",true);app->poll_interval_ms=z_desktop_opt_int(options,"pollIntervalMs",16);app->next_window_id=1;const char*backend=z_desktop_opt_string(options,"backend","auto");const char*headless_env=getenv("ZUMBRA_DESKTOP_HEADLESS");app->headless=strcmp(backend,"headless")==0||(headless_env!=NULL&&(strcmp(headless_env,"1")==0||strcasecmp(headless_env,"true")==0));
#if defined(__linux__)
    if(!app->headless){if(!z_desktop_sdl_load())z_fatal("desktopApp: %s",z_desktop_sdl.error);if(z_desktop_sdl.SetAppMetadata!=NULL)z_desktop_sdl.SetAppMetadata(z_desktop_opt_string(options,"name","Zumbra"),z_desktop_opt_string(options,"version",""),z_desktop_opt_string(options,"identifier","dev.zumbra.app"));}
#else
    app->headless=true;
#endif
    if (app->headless) z_desktop_queue(app,"ready",0,z_desktop_dict1("backend",z_string("headless")));
    return z_desktop_object(ZV_DESKTOP_APP,ZK_DESKTOP_APP,app);
}
static ZValue z_desktop_window_new(ZDesktopAppNative*app,ZValue options){
    if (options.tag != ZV_DICT) z_fatal("desktopWindow options must be dictionary");
    if (app->closed) z_fatal("desktop app is closed");
    int64_t width=z_desktop_opt_int(options,"width",800),height=z_desktop_opt_int(options,"height",600);if(width<1||height<1)z_fatal("window width and height must be positive");ZDesktopWindowNative*w=(ZDesktopWindowNative*)z_alloc(sizeof(*w));memset(w,0,sizeof(*w));w->app=app;w->title=z_strdup(z_desktop_opt_string(options,"title","Zumbra"));w->width=width;w->height=height;w->pixel_width=width;w->pixel_height=height;w->scale=z_desktop_opt_float(options,"scale",1.0);if(w->scale<=0)w->scale=1.0;w->density=z_desktop_opt_bool(options,"highDPI",true)?w->scale:1.0;w->pixel_width=(int64_t)((double)width*w->density);w->pixel_height=(int64_t)((double)height*w->density);w->open=true;w->visible=!z_desktop_opt_bool(options,"hidden",false);w->fullscreen=z_desktop_opt_bool(options,"fullscreen",false);
#if defined(__linux__)
    if(!app->headless){uint64_t flags=0;if(w->fullscreen)flags|=0x1u;if(!w->visible)flags|=0x8u;if(z_desktop_opt_bool(options,"borderless",false))flags|=0x10u;if(z_desktop_opt_bool(options,"resizable",true))flags|=0x20u;if(z_desktop_opt_bool(options,"highDPI",true))flags|=0x2000u;if(z_desktop_opt_bool(options,"alwaysOnTop",false))flags|=0x8000u;if(z_desktop_opt_bool(options,"transparent",false))flags|=0x40000000u;SDL_Window*handle=z_desktop_sdl.CreateWindow(w->title,(int)width,(int)height,flags);if(handle==NULL)z_fatal("desktopWindow: %s",z_desktop_sdl.GetError());w->handle=handle;w->id=(int64_t)z_desktop_sdl.GetWindowID(handle);w->scale=z_desktop_sdl.GetWindowDisplayScale(handle);w->density=z_desktop_sdl.GetWindowPixelDensity(handle);int pixel_width=(int)width,pixel_height=(int)height;if(z_desktop_sdl.GetWindowSizeInPixels!=NULL&&z_desktop_sdl.GetWindowSizeInPixels(handle,&pixel_width,&pixel_height)){w->pixel_width=pixel_width;w->pixel_height=pixel_height;}const char*icon=z_desktop_opt_string(options,"icon","");if(*icon){SDL_Surface*surface=z_desktop_sdl_load_bmp(icon);if(surface==NULL)z_fatal("could not load BMP icon");if(!z_desktop_sdl.SetWindowIcon(handle,surface)){z_desktop_sdl.DestroySurface(surface);z_fatal("could not set window icon");}z_desktop_sdl.DestroySurface(surface);}}
#endif
    if (app->headless) w->id = app->next_window_id++;
    w->next = app->windows;
    app->windows = w;
    if (app->headless) {if(w->visible)z_desktop_queue(app,"window_shown",w->id,z_dict_from(NULL,0));z_desktop_queue(app,"window_resized",w->id,z_desktop_dict2("width",z_int(width),"height",z_int(height)));z_desktop_queue(app,"window_pixel_size_changed",w->id,z_desktop_dict3("width",z_int(w->pixel_width),"height",z_int(w->pixel_height),"scale",z_float(w->scale)));}return z_desktop_object(ZV_DESKTOP_WINDOW,ZK_DESKTOP_WINDOW,w);
}
static ZValue z_desktop_window_action(ZDesktopWindowNative*w,const char*name){if(!w->open)z_fatal("window is closed");
#if defined(__linux__)
    if(!w->app->headless){bool ok=true;if(strcmp(name,"show")==0)ok=z_desktop_sdl.ShowWindow((SDL_Window*)w->handle);else if(strcmp(name,"hide")==0)ok=z_desktop_sdl.HideWindow((SDL_Window*)w->handle);else if(strcmp(name,"maximize")==0)ok=z_desktop_sdl.MaximizeWindow((SDL_Window*)w->handle);else if(strcmp(name,"minimize")==0)ok=z_desktop_sdl.MinimizeWindow((SDL_Window*)w->handle);else if(strcmp(name,"restore")==0)ok=z_desktop_sdl.RestoreWindow((SDL_Window*)w->handle);else if(strcmp(name,"focus")==0)ok=z_desktop_sdl.RaiseWindow((SDL_Window*)w->handle);if(!ok)z_fatal("desktop window action failed: %s",z_desktop_sdl.GetError());}
#endif
    if(w->app->headless){const char*event=strcmp(name,"show")==0?"window_shown":strcmp(name,"hide")==0?"window_hidden":strcmp(name,"maximize")==0?"window_maximized":strcmp(name,"minimize")==0?"window_minimized":strcmp(name,"restore")==0?"window_restored":"window_focus_gained";z_desktop_queue(w->app,event,w->id,z_dict_from(NULL,0));}return z_desktop_object(ZV_DESKTOP_WINDOW,ZK_DESKTOP_WINDOW,w);}

/* -------- process helpers -------- */
static ZValue z_desktop_process_spawn(const char*command,ZValue options){if(command==NULL||*command=='\0')z_fatal("desktopSpawn command cannot be empty");ZValue arg_value=z_desktop_dict_get(options,"args");size_t extra=(arg_value.tag==ZV_ARRAY?arg_value.as.array->len:0);char**argv=(char**)calloc(extra+2,sizeof(char*));if(argv==NULL)z_fatal("out of memory");argv[0]=(char*)command;for(size_t i=0;i<extra;i++){ZValue value=arg_value.as.array->items[i];if(value.tag!=ZV_STRING)z_fatal("desktopSpawn args must contain strings");argv[i+1]=(char*)value.as.s;}argv[extra+1]=NULL;pid_t pid=fork();if(pid<0){free(argv);z_fatal("desktopSpawn failed: %s",strerror(errno));}if(pid==0){const char*cwd=z_desktop_opt_string(options,"cwd","");if(*cwd&&chdir(cwd)!=0)_exit(126);ZValue env=z_desktop_dict_get(options,"env");if(env.tag==ZV_DICT){for(size_t i=0;i<env.as.dict->len;i++){ZValue k=env.as.dict->keys[i],v=env.as.dict->values[i];if(k.tag==ZV_STRING&&v.tag==ZV_STRING)setenv(k.as.s,v.as.s,1);}}execvp(command,argv);_exit(127);}free(argv);ZDesktopProcessNative*p=(ZDesktopProcessNative*)z_alloc(sizeof(*p));p->pid=pid;p->running=true;p->waited=false;p->exit_code=0;return z_desktop_object(ZV_DESKTOP_PROCESS,ZK_DESKTOP_PROCESS,p);}

/* -------- tray callback -------- */
#if defined(__linux__)
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
    if(strcmp(name,"desktopPoll")==0){z_expect_args(name,argc,2);ZDesktopAppNative*app=z_desktop_expect_app(args[0]);ZDesktopEventNative*event=z_desktop_next_event(app,z_as_i64(args[1]));if(event==NULL)return z_null();z_desktop_dispatch(app,event);return z_desktop_event_value(event);}
    if(strcmp(name,"desktopRun")==0){z_expect_args(name,argc,1);ZDesktopAppNative*app=z_desktop_expect_app(args[0]);if(app->closed)z_fatal("desktop app is closed");app->running=true;while(app->running&&!app->closed){ZDesktopEventNative*event=z_desktop_next_event(app,app->poll_interval_ms);if(event!=NULL)z_desktop_dispatch(app,event);}return z_bool(true);}
    if(strcmp(name,"desktopQuit")==0){z_expect_args(name,argc,1);z_desktop_expect_app(args[0])->running=false;return z_bool(true);}
    if(strcmp(name,"desktopRunning")==0){z_expect_args(name,argc,1);return z_bool(z_desktop_expect_app(args[0])->running);}
    if(strcmp(name,"desktopClose")==0){z_expect_args(name,argc,1);ZDesktopAppNative*app=z_desktop_expect_app(args[0]);if(!app->closed){app->running=false;app->closed=true;for(ZDesktopWindowNative*w=app->windows;w!=NULL;w=w->next)z_desktop_close_window_native(w);for(ZDesktopTrayNative*t=app->trays;t!=NULL;t=t->next){if(t->open){
#if defined(__linux__)
            if(!app->headless&&t->handle!=NULL&&z_desktop_sdl.DestroyTray!=NULL)z_desktop_sdl.DestroyTray((SDL_Tray*)t->handle);
#endif
            t->open=false;}}
#if defined(__linux__)
        if(!app->headless&&z_desktop_sdl.initialized){z_desktop_sdl.Quit();z_desktop_sdl.initialized=false;}
#endif
        }return z_bool(true);}
    if(strcmp(name,"desktopEmit")==0){z_expect_args(name,argc,2);ZDesktopAppNative*app=z_desktop_expect_app(args[0]);ZValue event=args[1];if(event.tag!=ZV_DICT)z_fatal("desktopEmit event must be dictionary");const char*type=z_desktop_opt_string(event,"type","custom");int64_t window_id=z_desktop_opt_int(event,"windowId",0);ZValue data=z_desktop_dict_get(event,"data");if(data.tag!=ZV_DICT)data=z_dict_from(NULL,0);z_desktop_queue(app,type,window_id,data);return z_bool(true);}
    if(strcmp(name,"desktopSetClipboard")==0){z_expect_args(name,argc,2);ZDesktopAppNative*app=z_desktop_expect_app(args[0]);const char*text=z_as_cstring(args[1]);if(app->headless){app->clipboard=z_strdup(text);z_desktop_queue(app,"clipboard_updated",0,z_dict_from(NULL,0));return z_bool(true);}
#if defined(__linux__)
        if (!z_desktop_sdl.SetClipboardText(text)) z_fatal("desktop clipboard: %s",z_desktop_sdl.GetError());
        return z_bool(true);
#else
        return z_bool(false);
#endif
    }
    if(strcmp(name,"desktopClipboard")==0){z_expect_args(name,argc,1);ZDesktopAppNative*app=z_desktop_expect_app(args[0]);if(app->headless)return z_string(app->clipboard==NULL?"":app->clipboard);
#if defined(__linux__)
        char*text=z_desktop_sdl.GetClipboardText();if(text==NULL)z_fatal("desktop clipboard: %s",z_desktop_sdl.GetError());ZValue result=z_string(text);z_desktop_sdl.Free(text);return result;
#else
        return z_string("");
#endif
    }
    if(strcmp(name,"desktopPickFile")==0){z_expect_args(name,argc,2);return z_desktop_pick_file_native(args[1],false);}
    if(strcmp(name,"desktopPickFolder")==0){z_expect_args(name,argc,2);return z_desktop_pick_file_native(args[1],true);}
    if(strcmp(name,"desktopNotify")==0){z_expect_args(name,argc,2);ZDesktopAppNative*app=z_desktop_expect_app(args[0]);const char*title=z_desktop_opt_string(args[1],"title","Zumbra"),*body=z_desktop_opt_string(args[1],"body","");if(app->headless){z_desktop_queue(app,"notification",0,z_desktop_dict2("title",z_string(title),"body",z_string(body)));return z_bool(true);}if(!z_desktop_command_exists("notify-send"))z_fatal("notify-send is required for notifications");z_desktop_spawn_detached("notify-send",title,body);return z_bool(true);}
    if(strcmp(name,"desktopPaths")==0){z_expect_args(name,argc,1);(void)z_desktop_expect_app(args[0]);return z_desktop_paths_native();}
    if(strcmp(name,"desktopOpenExternal")==0){z_expect_args(name,argc,2);ZDesktopAppNative*app=z_desktop_expect_app(args[0]);const char*target=z_as_cstring(args[1]);if(app->headless){z_desktop_queue(app,"external_opened",0,z_desktop_dict1("target",z_string(target)));return z_bool(true);}if(!z_desktop_command_exists("xdg-open"))z_fatal("xdg-open is required to open external resources");z_desktop_spawn_detached("xdg-open",target,NULL);return z_bool(true);}
    if(strcmp(name,"desktopTray")==0){z_expect_args(name,argc,2);ZDesktopAppNative*app=z_desktop_expect_app(args[0]);ZDesktopTrayNative*t=(ZDesktopTrayNative*)z_alloc(sizeof(*t));memset(t,0,sizeof(*t));t->app=app;t->open=true;
#if defined(__linux__)
        if(!app->headless){if(z_desktop_sdl.CreateTray==NULL||z_desktop_sdl.CreateTrayMenu==NULL)z_fatal("SDL3 tray APIs are unavailable");const char*icon=z_desktop_opt_string(args[1],"icon","");SDL_Surface*surface=z_desktop_sdl_load_bmp(icon);SDL_Tray*tray=z_desktop_sdl.CreateTray(surface,z_desktop_opt_string(args[1],"tooltip","Zumbra"));if(surface!=NULL&&z_desktop_sdl.DestroySurface!=NULL)z_desktop_sdl.DestroySurface(surface);if(tray==NULL)z_fatal("could not create system tray");SDL_TrayMenu*menu=z_desktop_sdl.CreateTrayMenu(tray);if(menu==NULL){z_desktop_sdl.DestroyTray(tray);z_fatal("could not create tray menu");}t->handle=tray;t->menu=menu;}
#endif
        t->next=app->trays;app->trays=t;if(app->headless)z_desktop_queue(app,"tray_created",0,z_dict_from(NULL,0));return z_desktop_object(ZV_DESKTOP_TRAY,ZK_DESKTOP_TRAY,t);}
    if(strcmp(name,"desktopTrayAdd")==0){z_expect_args(name,argc,4);ZDesktopTrayNative*t=z_desktop_expect_tray(args[0]);const char*id=z_as_cstring(args[1]),*label=z_as_cstring(args[2]);if(!t->open)z_fatal("tray is closed");char key[320];snprintf(key,sizeof(key),"tray:%s",id);z_desktop_add_handler(&t->app->handlers,key,args[3]);
#if defined(__linux__)
        if(!t->app->headless){SDL_TrayEntry*entry=z_desktop_sdl.InsertTrayEntryAt((SDL_TrayMenu*)t->menu,-1,label,1u);if(entry==NULL)z_fatal("could not create tray entry");ZDesktopTrayCallback*callback=(ZDesktopTrayCallback*)z_alloc(sizeof(*callback));callback->app=t->app;callback->id=z_strdup(id);z_desktop_sdl.SetTrayEntryCallback(entry,z_desktop_tray_callback,callback);}
#endif
        return args[0];}
    if(strcmp(name,"desktopTrayTooltip")==0){z_expect_args(name,argc,2);ZDesktopTrayNative*t=z_desktop_expect_tray(args[0]);if(!t->open)z_fatal("tray is closed");
#if defined(__linux__)
        if(!t->app->headless&&z_desktop_sdl.SetTrayTooltip!=NULL)z_desktop_sdl.SetTrayTooltip((SDL_Tray*)t->handle,z_as_cstring(args[1]));
#endif
        return args[0];}
    if(strcmp(name,"desktopTrayClose")==0||strcmp(name,"desktopTrayOpen")==0){z_expect_args(name,argc,1);ZDesktopTrayNative*t=z_desktop_expect_tray(args[0]);if(strcmp(name,"desktopTrayOpen")==0)return z_bool(t->open);if(t->open){
#if defined(__linux__)
            if(!t->app->headless&&t->handle!=NULL&&z_desktop_sdl.DestroyTray!=NULL)z_desktop_sdl.DestroyTray((SDL_Tray*)t->handle);
#endif
            t->open=false;}return z_bool(true);}
    if(strcmp(name,"desktopSpawn")==0){z_expect_args(name,argc,2);return z_desktop_process_spawn(z_as_cstring(args[0]),args[1]);}
    if(strcmp(name,"desktopProcessWait")==0){z_expect_args(name,argc,1);ZDesktopProcessNative*p=z_desktop_expect_process(args[0]);if(!p->waited){int status=0;if(waitpid(p->pid,&status,0)<0)z_fatal("waitpid failed: %s",strerror(errno));p->waited=true;p->running=false;p->exit_code=WIFEXITED(status)?WEXITSTATUS(status):(WIFSIGNALED(status)?128+WTERMSIG(status):-1);}return z_int(p->exit_code);}
    if(strcmp(name,"desktopProcessKill")==0){z_expect_args(name,argc,1);ZDesktopProcessNative*p=z_desktop_expect_process(args[0]);if(p->running&&kill(p->pid,SIGTERM)!=0&&errno!=ESRCH)z_fatal("could not kill process: %s",strerror(errno));return z_bool(true);}
    if(strcmp(name,"desktopProcessRunning")==0){z_expect_args(name,argc,1);ZDesktopProcessNative*p=z_desktop_expect_process(args[0]);if(p->running){int status=0;pid_t state=waitpid(p->pid,&status,WNOHANG);if(state==p->pid){p->running=false;p->waited=true;p->exit_code=WIFEXITED(status)?WEXITSTATUS(status):-1;}}return z_bool(p->running);}
    if(strcmp(name,"desktopProcessId")==0){z_expect_args(name,argc,1);return z_int(z_desktop_expect_process(args[0])->pid);}
    if(strncmp(name,"desktopWindow",13)==0){ZDesktopWindowNative*w=z_desktop_expect_window(args[0]);if(strcmp(name,"desktopWindowShow")==0){z_expect_args(name,argc,1);return z_desktop_window_action(w,"show");}if(strcmp(name,"desktopWindowHide")==0){z_expect_args(name,argc,1);return z_desktop_window_action(w,"hide");}if(strcmp(name,"desktopWindowClose")==0){z_expect_args(name,argc,1);z_desktop_close_window_native(w);return args[0];}if(strcmp(name,"desktopWindowOpen")==0){z_expect_args(name,argc,1);return z_bool(w->open);}if(strcmp(name,"desktopWindowId")==0){z_expect_args(name,argc,1);return z_int(w->id);}if(strcmp(name,"desktopWindowTitle")==0){z_expect_args(name,argc,1);
#if defined(__linux__)
            if(!w->app->headless&&w->open)w->title=z_strdup(z_desktop_sdl.GetWindowTitle((SDL_Window*)w->handle));
#endif
            return z_string(w->title);}if(strcmp(name,"desktopWindowSetTitle")==0){z_expect_args(name,argc,2);const char*title=z_as_cstring(args[1]);w->title=z_strdup(title);
#if defined(__linux__)
            if(!w->app->headless&&!z_desktop_sdl.SetWindowTitle((SDL_Window*)w->handle,title))z_fatal("could not set window title");
#endif
            return args[0];}if(strcmp(name,"desktopWindowSize")==0||strcmp(name,"desktopWindowPixelSize")==0){z_expect_args(name,argc,1);int64_t a=w->width,b=w->height;
#if defined(__linux__)
            if(!w->app->headless){int x=0,y=0;bool ok=strcmp(name,"desktopWindowPixelSize")==0?z_desktop_sdl.GetWindowSizeInPixels((SDL_Window*)w->handle,&x,&y):z_desktop_sdl.GetWindowSize((SDL_Window*)w->handle,&x,&y);if(ok){a=x;b=y;}}
#endif
            return z_desktop_dict2("width",z_int(a),"height",z_int(b));}if(strcmp(name,"desktopWindowSetSize")==0){z_expect_args(name,argc,3);int64_t width=z_as_i64(args[1]),height=z_as_i64(args[2]);if(width<1||height<1)z_fatal("window width and height must be positive");w->width=width;w->height=height;w->pixel_width=(int64_t)(width*w->density);w->pixel_height=(int64_t)(height*w->density);
#if defined(__linux__)
            if(!w->app->headless&&!z_desktop_sdl.SetWindowSize((SDL_Window*)w->handle,(int)width,(int)height))z_fatal("could not resize window");
#endif
            if (w->app->headless) z_desktop_queue(w->app,"window_resized",w->id,z_desktop_dict2("width",z_int(width),"height",z_int(height)));
            return args[0];
        }
        if(strcmp(name,"desktopWindowPosition")==0){z_expect_args(name,argc,1);int64_t x=w->x,y=w->y;
#if defined(__linux__)
            if(!w->app->headless){int a=0,b=0;if(z_desktop_sdl.GetWindowPosition((SDL_Window*)w->handle,&a,&b)){x=a;y=b;}}
#endif
            return z_desktop_dict2("x",z_int(x),"y",z_int(y));}if(strcmp(name,"desktopWindowSetPosition")==0){z_expect_args(name,argc,3);w->x=z_as_i64(args[1]);w->y=z_as_i64(args[2]);
#if defined(__linux__)
            if(!w->app->headless&&!z_desktop_sdl.SetWindowPosition((SDL_Window*)w->handle,(int)w->x,(int)w->y))z_fatal("could not move window");
#endif
            if (w->app->headless) z_desktop_queue(w->app,"window_moved",w->id,z_desktop_dict2("x",z_int(w->x),"y",z_int(w->y)));
            return args[0];
        }
        if(strcmp(name,"desktopWindowFullscreen")==0){z_expect_args(name,argc,1);
#if defined(__linux__)
            if(!w->app->headless)w->fullscreen=(z_desktop_sdl.GetWindowFlags((SDL_Window*)w->handle)&0x1u)!=0;
#endif
            return z_bool(w->fullscreen);}if(strcmp(name,"desktopWindowSetFullscreen")==0){z_expect_args(name,argc,2);w->fullscreen=z_as_bool(args[1]);
#if defined(__linux__)
            if(!w->app->headless&&!z_desktop_sdl.SetWindowFullscreen((SDL_Window*)w->handle,w->fullscreen))z_fatal("could not change fullscreen");
#endif
            if (w->app->headless) z_desktop_queue(w->app,w->fullscreen?"window_enter_fullscreen":"window_leave_fullscreen",w->id,z_dict_from(NULL,0));
            return args[0];
        }
        if(strcmp(name,"desktopWindowMaximize")==0){z_expect_args(name,argc,1);return z_desktop_window_action(w,"maximize");}if(strcmp(name,"desktopWindowMinimize")==0){z_expect_args(name,argc,1);return z_desktop_window_action(w,"minimize");}if(strcmp(name,"desktopWindowRestore")==0){z_expect_args(name,argc,1);return z_desktop_window_action(w,"restore");}if(strcmp(name,"desktopWindowFocus")==0){z_expect_args(name,argc,1);return z_desktop_window_action(w,"focus");}if(strcmp(name,"desktopWindowDisplayScale")==0){z_expect_args(name,argc,1);
#if defined(__linux__)
            if(!w->app->headless)w->scale=z_desktop_sdl.GetWindowDisplayScale((SDL_Window*)w->handle);
#endif
            return z_float(w->scale);}if(strcmp(name,"desktopWindowPixelDensity")==0){z_expect_args(name,argc,1);
#if defined(__linux__)
            if(!w->app->headless)w->density=z_desktop_sdl.GetWindowPixelDensity((SDL_Window*)w->handle);
#endif
            return z_float(w->density);}if(strcmp(name,"desktopWindowSetIcon")==0){z_expect_args(name,argc,2);
#if defined(__linux__)
            if(!w->app->headless){if(z_desktop_sdl.IOFromFile==NULL||z_desktop_sdl.LoadBMP_IO==NULL||z_desktop_sdl.SetWindowIcon==NULL)z_fatal("SDL3 BMP icon APIs are unavailable");SDL_Surface*surface=z_desktop_sdl_load_bmp(z_as_cstring(args[1]));if(surface==NULL)z_fatal("could not load BMP icon");bool ok=z_desktop_sdl.SetWindowIcon((SDL_Window*)w->handle,surface);z_desktop_sdl.DestroySurface(surface);if(!ok)z_fatal("could not set window icon");}
#endif
            return args[0];}}
    if (handled != NULL) *handled = false;
    return z_null();
}

static void z_desktop_runtime_init(void) {
#if defined(__linux__)
    memset(&z_desktop_sdl,0,sizeof(z_desktop_sdl));
#endif
}
static void z_desktop_runtime_shutdown(void) {
#if defined(__linux__)
    if(z_desktop_sdl.initialized&&z_desktop_sdl.Quit!=NULL)z_desktop_sdl.Quit();
    if(z_desktop_sdl.lib!=NULL)dlclose(z_desktop_sdl.lib);
    memset(&z_desktop_sdl,0,sizeof(z_desktop_sdl));
#endif
}

#endif
