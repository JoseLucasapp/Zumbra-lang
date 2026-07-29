#if defined(ZUMBRA_ENABLE_UI)

/* Z14 retained-mode GUI toolkit. It shares Z13's SDL3 loader and uses a
 * deterministic headless layout/render path for automated tests. */

typedef struct ZUINodeNative ZUINodeNative;
typedef struct ZUIStateNative ZUIStateNative;
typedef struct ZUIThemeNative ZUIThemeNative;
typedef struct ZUIContextNative ZUIContextNative;
typedef struct ZUIBindingNative ZUIBindingNative;
typedef struct ZUISubscriberNative ZUISubscriberNative;
typedef struct ZUIReverseBindingNative ZUIReverseBindingNative;

typedef struct { double x,y,width,height; } ZUIRectNative;
typedef struct { ZValue receiver; const char *builtin; } ZUIMethodNative;

struct ZUINodeNative {
    char *id;
    char *kind;
    ZValue props;
    ZValue children;
    ZUINodeNative *parent;
    ZUIContextNative *context;
    ZUIReverseBindingNative *bindings;
    ZUIRectNative bounds;
    ZUIRectNative content_bounds;
    double scroll_offset_y;
    double content_height;
    bool visible;
    bool enabled;
};
struct ZUIBindingNative { ZUINodeNative *node; char *property; ZUIBindingNative *next; };
struct ZUIReverseBindingNative { char *property; ZUIStateNative *state; ZUIReverseBindingNative *next; };
struct ZUISubscriberNative { ZValue handler; ZUISubscriberNative *next; };
struct ZUIStateNative { ZValue value; ZUIBindingNative *bindings; ZUISubscriberNative *subscribers; };
struct ZUIThemeNative { char *name; ZValue values; };
struct ZUIContextNative {
    ZDesktopAppNative *app;
    ZDesktopWindowNative *window;
    ZUINodeNative *root;
    ZUIThemeNative *theme;
    char *focus_id;
    char *hover_id;
    bool dirty;
    bool closed;
    struct ZUIContextNative *next;
};

static int64_t z_ui_next_id=1;
static double z_ui_prop_number(ZValue props,const char*key,double fallback);
static const char*z_ui_prop_string(ZValue props,const char*key,const char*fallback);
static double z_ui_theme_number(ZUIThemeNative*t,const char*key,double fallback);
static const char*z_ui_theme_string(ZUIThemeNative*t,const char*key,const char*fallback);
#if defined(ZUMBRA_DESKTOP_SDL3)
typedef struct TTF_Font TTF_Font;
typedef struct { uint8_t r,g,b,a; } ZUITTFColor;
typedef bool (*ZUI_TTF_Init)(void);
typedef void (*ZUI_TTF_Quit)(void);
typedef TTF_Font *(*ZUI_TTF_OpenFont)(const char*,float);
typedef void (*ZUI_TTF_CloseFont)(TTF_Font*);
typedef void (*ZUI_TTF_SetFontStyle)(TTF_Font*,uint32_t);
typedef bool (*ZUI_TTF_GetStringSize)(TTF_Font*,const char*,size_t,int*,int*);
typedef bool (*ZUI_TTF_GetStringSizeWrapped)(TTF_Font*,const char*,size_t,int,int*,int*);
typedef SDL_Surface *(*ZUI_TTF_RenderText_Blended)(TTF_Font*,const char*,size_t,ZUITTFColor);
typedef SDL_Surface *(*ZUI_TTF_RenderText_Blended_Wrapped)(TTF_Font*,const char*,size_t,ZUITTFColor,int);
typedef struct ZUIFontCacheNative { char path[PATH_MAX]; float size; uint32_t style; TTF_Font *font; struct ZUIFontCacheNative *next; } ZUIFontCacheNative;
typedef struct {
    void *lib; bool initialized;
    ZUI_TTF_Init Init; ZUI_TTF_Quit Quit; ZUI_TTF_OpenFont OpenFont; ZUI_TTF_CloseFont CloseFont;
    ZUI_TTF_SetFontStyle SetFontStyle; ZUI_TTF_GetStringSize GetStringSize; ZUI_TTF_GetStringSizeWrapped GetStringSizeWrapped;
    ZUI_TTF_RenderText_Blended RenderTextBlended; ZUI_TTF_RenderText_Blended_Wrapped RenderTextBlendedWrapped;
    ZUIFontCacheNative *fonts;
} ZUITTFRuntime;
static ZUITTFRuntime z_ui_ttf;
static bool z_ui_contains_ci(const char*text,const char*needle){if(text==NULL||needle==NULL||*needle=='\0')return false;size_t n=strlen(needle);for(const char*p=text;*p;p++){size_t i=0;while(i<n&&p[i]&&tolower((unsigned char)p[i])==tolower((unsigned char)needle[i]))i++;if(i==n)return true;}return false;}
static bool z_ui_file_exists(const char*path){return path!=NULL&&*path!='\0'&&z_desktop_access(path,4)==0;}
static bool z_ui_font_bold(const char*weight){if(weight==NULL)return false;if(strcasecmp(weight,"bold")==0||strcasecmp(weight,"semibold")==0)return true;char*end=NULL;long v=strtol(weight,&end,10);return end!=weight&&v>=600;}
static bool z_ui_font_italic(const char*style){return style!=NULL&&(strcasecmp(style,"italic")==0||strcasecmp(style,"oblique")==0);}
static const char*z_ui_resolve_font(const char*family,const char*explicit_path,const char*weight,const char*style){
    static char selected[PATH_MAX];selected[0]='\0';const char*env=getenv("ZUMBRA_UI_FONT");
    if(z_ui_file_exists(explicit_path)){snprintf(selected,sizeof(selected),"%s",explicit_path);return selected;}
    if(family&&strchr(family,'/')&&z_ui_file_exists(family)){snprintf(selected,sizeof(selected),"%s",family);return selected;}
    if(z_ui_file_exists(env)){snprintf(selected,sizeof(selected),"%s",env);return selected;}
    bool mono=family&&(z_ui_contains_ci(family,"mono")||z_ui_contains_ci(family,"code"));bool serif=family&&z_ui_contains_ci(family,"serif")&&!mono;bool bold=z_ui_font_bold(weight),italic=z_ui_font_italic(style);
    const char*candidates[32];size_t n=0;
#if defined(_WIN32)
    static char win_fonts[PATH_MAX];const char*windir=getenv("WINDIR");if(windir==NULL)windir="C:/Windows";snprintf(win_fonts,sizeof(win_fonts),"%s/Fonts",windir);
    static char win_paths[8][PATH_MAX];const char*files[8]={mono?"consola.ttf":serif?"times.ttf":"segoeui.ttf",mono?"cour.ttf":serif?"timesbd.ttf":"segoeuib.ttf","arial.ttf","arialbd.ttf","calibri.ttf","calibrib.ttf",NULL,NULL};
    for(size_t i=0;i<8&&files[i];i++){snprintf(win_paths[i],sizeof(win_paths[i]),"%s/%s",win_fonts,files[i]);candidates[n++]=win_paths[i];}
#elif defined(__APPLE__)
    if(mono){candidates[n++]="/System/Library/Fonts/SFNSMono.ttf";candidates[n++]="/System/Library/Fonts/Monaco.ttf";}else if(serif){candidates[n++]="/System/Library/Fonts/NewYork.ttf";candidates[n++]="/System/Library/Fonts/Supplemental/Times New Roman.ttf";}else{candidates[n++]="/System/Library/Fonts/SFNS.ttf";candidates[n++]="/System/Library/Fonts/Supplemental/Arial.ttf";}
#else
    if(mono){if(bold&&italic)candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSansMono-BoldOblique.ttf";else if(bold)candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSansMono-Bold.ttf";else if(italic)candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSansMono-Oblique.ttf";candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf";candidates[n++]="/usr/share/fonts/truetype/liberation2/LiberationMono-Regular.ttf";}
    else if(serif){if(bold&&italic)candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSerif-BoldItalic.ttf";else if(bold)candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSerif-Bold.ttf";else if(italic)candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSerif-Italic.ttf";candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSerif.ttf";candidates[n++]="/usr/share/fonts/truetype/liberation2/LiberationSerif-Regular.ttf";}
    else {if(bold&&italic)candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSans-BoldOblique.ttf";else if(bold)candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf";else if(italic)candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSans-Oblique.ttf";candidates[n++]="/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf";candidates[n++]="/usr/share/fonts/truetype/liberation2/LiberationSans-Regular.ttf";candidates[n++]="/usr/share/fonts/opentype/noto/NotoSans-Regular.ttf";}
#endif
    for(size_t i=0;i<n;i++) {
        if(z_ui_file_exists(candidates[i])) {
            snprintf(selected,sizeof(selected),"%s",candidates[i]);
            return selected;
        }
    }
    return NULL;
}
static bool z_ui_ttf_load(void){
    if(z_ui_ttf.initialized) return true;
    if(z_ui_ttf.lib==NULL) {
#if defined(_WIN32)
        const char*names[]={"SDL3_ttf.dll","libSDL3_ttf.dll",NULL};
#elif defined(__APPLE__)
        const char*names[]={"@executable_path/../Frameworks/libSDL3_ttf.0.dylib","@executable_path/../Frameworks/libSDL3_ttf.dylib","libSDL3_ttf.0.dylib","libSDL3_ttf.dylib",NULL};
#else
        const char*names[]={"libSDL3_ttf.so.0","libSDL3_ttf.so",NULL};
#endif
        for(size_t i=0;names[i];i++) {
            z_ui_ttf.lib=z_desktop_library_open(names[i]);
            if(z_ui_ttf.lib) break;
        }
    }
    if(z_ui_ttf.lib==NULL) return false;
#define ZUI_TTF_REQ(field,symbol,type) do{z_ui_ttf.field=(type)z_desktop_library_symbol(z_ui_ttf.lib,symbol);if(z_ui_ttf.field==NULL)return false;}while(0)
#define ZUI_TTF_OPT(field,symbol,type) do{z_ui_ttf.field=(type)z_desktop_library_symbol(z_ui_ttf.lib,symbol);}while(0)
    ZUI_TTF_REQ(Init,"TTF_Init",ZUI_TTF_Init);ZUI_TTF_REQ(Quit,"TTF_Quit",ZUI_TTF_Quit);ZUI_TTF_REQ(OpenFont,"TTF_OpenFont",ZUI_TTF_OpenFont);ZUI_TTF_REQ(CloseFont,"TTF_CloseFont",ZUI_TTF_CloseFont);ZUI_TTF_OPT(SetFontStyle,"TTF_SetFontStyle",ZUI_TTF_SetFontStyle);ZUI_TTF_REQ(GetStringSize,"TTF_GetStringSize",ZUI_TTF_GetStringSize);ZUI_TTF_OPT(GetStringSizeWrapped,"TTF_GetStringSizeWrapped",ZUI_TTF_GetStringSizeWrapped);ZUI_TTF_REQ(RenderTextBlended,"TTF_RenderText_Blended",ZUI_TTF_RenderText_Blended);ZUI_TTF_OPT(RenderTextBlendedWrapped,"TTF_RenderText_Blended_Wrapped",ZUI_TTF_RenderText_Blended_Wrapped);
#undef ZUI_TTF_REQ
#undef ZUI_TTF_OPT
    if(!z_ui_ttf.Init()) return false;
    z_ui_ttf.initialized=true;
    return true;
}
static uint32_t z_ui_font_flags(const char*weight,const char*style){uint32_t flags=0;if(z_ui_font_bold(weight))flags|=0x01u;if(z_ui_font_italic(style))flags|=0x02u;return flags;}
static TTF_Font*z_ui_font(const char*family,const char*path,double size,const char*weight,const char*style){if(size<1)size=14;if(!z_ui_ttf_load())return NULL;const char*resolved=z_ui_resolve_font(family,path,weight,style);if(resolved==NULL)return NULL;uint32_t flags=z_ui_font_flags(weight,style);for(ZUIFontCacheNative*c=z_ui_ttf.fonts;c;c=c->next)if(c->size==(float)size&&c->style==flags&&strcmp(c->path,resolved)==0)return c->font;TTF_Font*font=z_ui_ttf.OpenFont(resolved,(float)size);if(font==NULL)return NULL;if(z_ui_ttf.SetFontStyle)z_ui_ttf.SetFontStyle(font,flags);ZUIFontCacheNative*c=(ZUIFontCacheNative*)calloc(1,sizeof(*c));if(c==NULL){z_ui_ttf.CloseFont(font);return NULL;}snprintf(c->path,sizeof(c->path),"%s",resolved);c->size=(float)size;c->style=flags;c->font=font;c->next=z_ui_ttf.fonts;z_ui_ttf.fonts=c;return font;}
static size_t z_ui_utf8_count(const char*text){size_t count=0;if(text==NULL)return 0;for(const unsigned char*p=(const unsigned char*)text;*p;p++)if((*p&0xC0u)!=0x80u)count++;return count;}
static void z_ui_measure_text_native(const char*text,ZValue props,ZUIThemeNative*t,double wrap,double*out_w,double*out_h){if(text==NULL)text="";double size=z_ui_prop_number(props,"fontSize",z_ui_theme_number(t,"fontSize",14));const char*family=z_ui_prop_string(props,"fontFamily",z_ui_theme_string(t,"fontFamily","sans"));const char*path=z_ui_prop_string(props,"fontPath",z_ui_theme_string(t,"fontPath",""));const char*weight=z_ui_prop_string(props,"fontWeight",z_ui_theme_string(t,"fontWeight","normal"));const char*style=z_ui_prop_string(props,"fontStyle",z_ui_theme_string(t,"fontStyle","normal"));TTF_Font*font=z_ui_font(family,path,size,weight,style);int w=0,h=0;bool ok=false;if(font){if(wrap>0&&z_ui_ttf.GetStringSizeWrapped)ok=z_ui_ttf.GetStringSizeWrapped(font,text,0,(int)wrap,&w,&h);else ok=z_ui_ttf.GetStringSize(font,text,0,&w,&h);}if(!ok){w=(int)((double)z_ui_utf8_count(text)*size*0.58);h=(int)(size*z_ui_prop_number(props,"lineHeight",z_ui_theme_number(t,"lineHeight",1.25)));if(wrap>0&&w>(int)wrap){int lines=(w+(int)wrap-1)/(int)wrap;w=(int)wrap;h*=lines;}}if(out_w)*out_w=w;if(out_h)*out_h=h;}
static void z_ui_ttf_shutdown(void){ZUIFontCacheNative*c=z_ui_ttf.fonts;while(c){ZUIFontCacheNative*n=c->next;if(c->font&&z_ui_ttf.CloseFont)z_ui_ttf.CloseFont(c->font);free(c);c=n;}z_ui_ttf.fonts=NULL;if(z_ui_ttf.initialized&&z_ui_ttf.Quit)z_ui_ttf.Quit();z_ui_ttf.initialized=false;if(z_ui_ttf.lib){z_desktop_library_close(z_ui_ttf.lib);z_ui_ttf.lib=NULL;}}
#else
static void z_ui_measure_text_native(const char*text,ZValue props,ZUIThemeNative*t,double wrap,double*out_w,double*out_h){double size=z_ui_prop_number(props,"fontSize",z_ui_theme_number(t,"fontSize",14));size_t count=text?strlen(text):0;double w=(double)count*size*0.58,h=size*1.25;if(wrap>0&&w>wrap){double lines=ceil(w/wrap);w=wrap;h*=lines;}if(out_w)*out_w=w;if(out_h)*out_h=h;}
static void z_ui_ttf_shutdown(void){}
#endif
static void z_ui_runtime_init(void){
    z_ui_next_id=1;
#if defined(ZUMBRA_DESKTOP_SDL3)
    memset(&z_ui_ttf,0,sizeof(z_ui_ttf));
#endif
}
static void z_ui_runtime_shutdown(void){z_ui_ttf_shutdown();}
static ZValue z_ui_value(ZTag tag,ZKind kind,void*p){ZValue v=z_value(tag,kind);v.as.p=p;return v;}
static ZUINodeNative*z_ui_expect_node(ZValue v){if(v.tag!=ZV_UI_NODE||v.as.p==NULL)z_fatal("expected UINode");return(ZUINodeNative*)v.as.p;}
static ZUIStateNative*z_ui_expect_state(ZValue v){if(v.tag!=ZV_UI_STATE||v.as.p==NULL)z_fatal("expected UIState");return(ZUIStateNative*)v.as.p;}
static ZUIThemeNative*z_ui_expect_theme(ZValue v){if(v.tag!=ZV_UI_THEME||v.as.p==NULL)z_fatal("expected UITheme");return(ZUIThemeNative*)v.as.p;}
static ZUIContextNative*z_ui_expect_context(ZValue v){if(v.tag!=ZV_UI_CONTEXT||v.as.p==NULL)z_fatal("expected UIContext");return(ZUIContextNative*)v.as.p;}
static double z_ui_number(ZValue v,double fallback){if(v.tag==ZV_FLOAT)return v.as.f;if(v.tag==ZV_INT)return(double)v.as.i;if(v.tag==ZV_UINT)return(double)v.as.u;return fallback;}
static double z_ui_prop_number(ZValue props,const char*key,double fallback){return z_ui_number(z_desktop_dict_get(props,key),fallback);}
static const char*z_ui_prop_string(ZValue props,const char*key,const char*fallback){return z_desktop_opt_string(props,key,fallback);}
static bool z_ui_prop_bool(ZValue props,const char*key,bool fallback){return z_desktop_opt_bool(props,key,fallback);}
static bool z_ui_scrollable_y(ZUINodeNative*n){if(n==NULL)return false;if(z_ui_prop_bool(n->props,"scrollY",false))return true;const char*overflow=z_ui_prop_string(n->props,"overflowY",z_ui_prop_string(n->props,"overflow",""));return strcasecmp(overflow,"scroll")==0||strcasecmp(overflow,"auto")==0;}
static double z_ui_clamp(double value,double minimum,double maximum){if(value<minimum)return minimum;if(value>maximum)return maximum;return value;}
static bool z_ui_rect_intersection(ZUIRectNative a,ZUIRectNative b,ZUIRectNative*out){double x1=fmax(a.x,b.x),y1=fmax(a.y,b.y),x2=fmin(a.x+a.width,b.x+b.width),y2=fmin(a.y+a.height,b.y+b.height);if(x2<=x1||y2<=y1)return false;if(out)*out=(ZUIRectNative){x1,y1,x2-x1,y2-y1};return true;}
static bool z_ui_point_in_rect(double x,double y,ZUIRectNative b){return x>=b.x&&y>=b.y&&x<=b.x+b.width&&y<=b.y+b.height;}
static void z_ui_dict_set(ZValue dict,const char*key,ZValue value){z_set_index(dict,z_string(key),value);}
static ZValue z_ui_theme_values(bool dark){
    ZValue pairs[18]={
        z_pair(z_string("background"),z_string(dark?"#141822":"#f5f7fb")),
        z_pair(z_string("surface"),z_string(dark?"#202633":"#ffffff")),
        z_pair(z_string("surfaceAlt"),z_string(dark?"#2a3242":"#eef2f7")),
        z_pair(z_string("text"),z_string(dark?"#f2f5fa":"#172033")),
        z_pair(z_string("muted"),z_string(dark?"#a9b4c7":"#697386")),
        z_pair(z_string("primary"),z_string("#3867e8")),
        z_pair(z_string("primaryText"),z_string("#ffffff")),
        z_pair(z_string("border"),z_string(dark?"#465066":"#cfd6e2")),
        z_pair(z_string("danger"),z_string("#c73737")),
        z_pair(z_string("focus"),z_string("#6e95ff")),
        z_pair(z_string("radius"),z_int(6)),
        z_pair(z_string("fontFamily"),z_string("sans")),
        z_pair(z_string("fontPath"),z_string("")),
        z_pair(z_string("fontWeight"),z_string("normal")),
        z_pair(z_string("fontStyle"),z_string("normal")),
        z_pair(z_string("fontSize"),z_int(14)),
        z_pair(z_string("lineHeight"),z_float(1.25)),
        z_pair(z_string("spacing"),z_int(8))
    };
    ZValue result=z_dict_from(pairs,18);
    z_ui_dict_set(result,"controlHeight",z_int(36));
    return result;
}
static ZUIThemeNative*z_ui_theme_new(ZValue source){
    ZUIThemeNative*t=(ZUIThemeNative*)z_alloc(sizeof(*t));memset(t,0,sizeof(*t));
    if(source.tag==ZV_STRING){t->name=z_strdup(source.as.s);t->values=z_ui_theme_values(strcasecmp(source.as.s,"dark")==0);}
    else if(source.tag==ZV_DICT){t->name=z_strdup("custom");t->values=z_ui_theme_values(false);for(size_t i=0;i<source.as.dict->len;i++)z_set_index(t->values,source.as.dict->keys[i],source.as.dict->values[i]);}
    else { z_fatal("uiTheme expects string or dictionary"); }
    return t;
}
static ZUINodeNative*z_ui_node_new(const char*kind,ZValue props,ZValue children){
    if(props.tag!=ZV_DICT) { z_fatal("UI node props must be dictionary"); }
    if(children.tag!=ZV_ARRAY) { z_fatal("UI node children must be array"); }
    ZValue nested=z_desktop_dict_get(props,"options");if(nested.tag==ZV_DICT){for(size_t i=0;i<nested.as.dict->len;i++){ZValue key=nested.as.dict->keys[i];if(key.tag==ZV_STRING&&z_desktop_dict_get(props,key.as.s).tag==ZV_NULL)z_set_index(props,key,nested.as.dict->values[i]);}}
    ZUINodeNative*n=(ZUINodeNative*)z_alloc(sizeof(*n));memset(n,0,sizeof(*n));n->kind=z_strdup(kind);n->props=props;n->children=children;n->visible=z_ui_prop_bool(props,"visible",true);n->enabled=!z_ui_prop_bool(props,"disabled",false);
    const char*id=z_ui_prop_string(props,"id","");char generated[64];if(*id=='\0'){snprintf(generated,sizeof(generated),"ui-%" PRId64,z_ui_next_id++);id=generated;}n->id=z_strdup(id);
    for(size_t i=0;i<children.as.array->len;i++){ZUINodeNative*c=z_ui_expect_node(children.as.array->items[i]);c->parent=n;}return n;
}
static void z_ui_attach_tree(ZUIContextNative*ctx,ZUINodeNative*n){if(n==NULL)return;n->context=ctx;for(size_t i=0;i<n->children.as.array->len;i++)z_ui_attach_tree(ctx,z_ui_expect_node(n->children.as.array->items[i]));}
static double z_ui_theme_number(ZUIThemeNative*t,const char*key,double fallback){return t?z_ui_prop_number(t->values,key,fallback):fallback;}
static const char*z_ui_theme_string(ZUIThemeNative*t,const char*key,const char*fallback){return t?z_ui_prop_string(t->values,key,fallback):fallback;}
static size_t z_ui_array_length(ZValue value){return value.tag==ZV_ARRAY?value.as.array->len:0;}
typedef struct {double left,top,right,bottom;} ZUIEdgesNative;
static ZUIEdgesNative z_ui_edges(ZValue props,const char*key,double fallback){
    ZUIEdgesNative e={fallback,fallback,fallback,fallback};ZValue all=z_desktop_dict_get(props,key);if(all.tag!=ZV_NULL){double v=z_ui_number(all,fallback);e=(ZUIEdgesNative){v,v,v,v};}
    char name[96];snprintf(name,sizeof(name),"%sLeft",key);e.left=z_ui_prop_number(props,name,e.left);snprintf(name,sizeof(name),"%sTop",key);e.top=z_ui_prop_number(props,name,e.top);snprintf(name,sizeof(name),"%sRight",key);e.right=z_ui_prop_number(props,name,e.right);snprintf(name,sizeof(name),"%sBottom",key);e.bottom=z_ui_prop_number(props,name,e.bottom);return e;
}
static const char*z_ui_props_text(ZValue props){
    const char*keys[]={"text","label","value","placeholder","title",NULL};
    for(size_t i=0;keys[i]!=NULL;i++){
        ZValue value=z_desktop_dict_get(props,keys[i]);
        if(value.tag==ZV_STRING&&value.as.s!=NULL&&*value.as.s!='\0')return value.as.s;
    }
    return "";
}
static double z_ui_natural_height(ZUINodeNative*n,ZUIThemeNative*t){
    double control=z_ui_theme_number(t,"controlHeight",36);
    double font=z_ui_prop_number(n->props,"fontSize",z_ui_theme_number(t,"fontSize",14));
    double text_width=0,text_height=0;
    z_ui_measure_text_native(z_ui_props_text(n->props),n->props,t,z_ui_prop_number(n->props,"wrapWidth",0),&text_width,&text_height);
    (void)text_width;
    if(strcmp(n->kind,"text")==0)return fmax(font+8,text_height+8);
    if(strcmp(n->kind,"textarea")==0){double rows=z_ui_prop_number(n->props,"rows",4),line_height=z_ui_prop_number(n->props,"lineHeight",z_ui_theme_number(t,"lineHeight",1.25));return fmax(rows*font*line_height+16,control);}
    if(strcmp(n->kind,"table")==0){size_t rows=z_ui_array_length(z_desktop_dict_get(n->props,"rows"));if(rows==0)rows=1;return(double)(rows+1)*control;}
    if(strcmp(n->kind,"list")==0||strcmp(n->kind,"tree")==0){size_t items=z_ui_array_length(z_desktop_dict_get(n->props,"items"));if(items==0)items=1;return(double)items*control;}
    if(strcmp(n->kind,"tabs")==0||strcmp(n->kind,"menu")==0)return control;
    if(strcmp(n->kind,"modal")==0)return z_ui_prop_number(n->props,"contentHeight",320);
    if(strcmp(n->kind,"tooltip")==0)return fmax(font+16,text_height+12);
    if(strcmp(n->kind,"progress")==0)return 20;
    if(strcmp(n->kind,"image")==0||strcmp(n->kind,"canvas")==0)return z_ui_prop_number(n->props,"height",160);
    if(strcmp(n->kind,"spacer")==0)return z_ui_prop_number(n->props,"size",8);
    return z_ui_prop_number(n->props,"height",control);
}
static double z_ui_natural_width(ZUINodeNative*n,ZUIThemeNative*t){
    double text_width=0,text_height=0;
    z_ui_measure_text_native(z_ui_props_text(n->props),n->props,t,0,&text_width,&text_height);
    (void)text_height;
    double width=text_width+24;if(width<48)width=48;
    if(strcmp(n->kind,"input")==0||strcmp(n->kind,"textarea")==0||strcmp(n->kind,"select")==0)if(width<180)width=180;
    if(strcmp(n->kind,"checkbox")==0||strcmp(n->kind,"radio")==0)width+=20;
    if(strcmp(n->kind,"image")==0||strcmp(n->kind,"canvas")==0)width=z_ui_prop_number(n->props,"width",240);
    if(strcmp(n->kind,"spacer")==0)width=z_ui_prop_number(n->props,"size",8);
    return width;
}
static void z_ui_layout(ZUINodeNative*n,ZUIRectNative available,ZUIThemeNative*t){
    if(n==NULL||!n->visible) return;
    ZUIEdgesNative margin=z_ui_edges(n->props,"margin",0);
    double x=available.x+margin.left,y=available.y+margin.top,width=available.width-margin.left-margin.right,height=available.height-margin.top-margin.bottom;
    ZValue wv=z_desktop_dict_get(n->props,"width"),hv=z_desktop_dict_get(n->props,"height");
    if(wv.tag!=ZV_NULL)width=z_ui_number(wv,width);if(hv.tag!=ZV_NULL)height=z_ui_number(hv,height);
    double minw=z_ui_prop_number(n->props,"minWidth",width),maxw=z_ui_prop_number(n->props,"maxWidth",width),minh=z_ui_prop_number(n->props,"minHeight",height),maxh=z_ui_prop_number(n->props,"maxHeight",height);
    if(width<minw)width=minw;if(width>maxw)width=maxw;if(height<minh)height=minh;if(height>maxh)height=maxh;if(width<0)width=0;if(height<0)height=0;
    if(hv.tag==ZV_NULL&&strcmp(n->kind,"row")!=0&&strcmp(n->kind,"column")!=0&&strcmp(n->kind,"container")!=0&&strcmp(n->kind,"modal")!=0){double natural=z_ui_natural_height(n,t);if(height>natural)height=natural;}
    if(strcmp(n->kind,"modal")==0){double mw=z_ui_prop_number(n->props,"contentWidth",480),mh=z_ui_prop_number(n->props,"contentHeight",320);if(mw>width)mw=width;if(mh>height)mh=height;x+=(width-mw)/2;y+=(height-mh)/2;width=mw;height=mh;}
    n->bounds=(ZUIRectNative){x,y,width,height};
    ZUIEdgesNative padding=z_ui_edges(n->props,"padding",0);
    ZUIRectNative inner={x+padding.left,y+padding.top,width-padding.left-padding.right,height-padding.top-padding.bottom};if(inner.width<0)inner.width=0;if(inner.height<0)inner.height=0;
    n->content_bounds=inner;n->content_height=inner.height;
    size_t count=n->children.as.array->len;if(count==0){n->scroll_offset_y=0;return;}
    const char*direction=n->kind;if(strcmp(direction,"row")!=0&&strcmp(direction,"column")!=0)direction=z_ui_prop_string(n->props,"direction","column");if(strcmp(n->kind,"modal")==0)direction="column";bool row=strcmp(direction,"row")==0;
    double gap=z_ui_prop_number(n->props,"gap",z_ui_theme_number(t,"spacing",8));if(gap<0)gap=0;
    double main=(row?inner.width:inner.height)-gap*(double)(count>0?count-1:0);if(main<0)main=0;double fixed=0,grow_total=0;
    for(size_t i=0;i<count;i++){ZUINodeNative*c=z_ui_expect_node(n->children.as.array->items[i]);double grow=z_ui_prop_number(c->props,"grow",0);if(grow>0)grow_total+=grow;else fixed+=row?z_ui_prop_number(c->props,"width",z_ui_natural_width(c,t)):z_ui_prop_number(c->props,"height",z_ui_natural_height(c,t));}
    double remaining=main-fixed;if(remaining<0)remaining=0;
    double content_height=inner.height;if(!row){double used=fixed+gap*(double)(count>0?count-1:0);if(grow_total>0)used+=remaining;if(used>content_height)content_height=used;}
    bool scrollable=z_ui_scrollable_y(n)&&!row;
    if(!scrollable)n->scroll_offset_y=0;else{double max_offset=fmax(0,content_height-inner.height);if(max_offset<=0.5)max_offset=0;n->scroll_offset_y=z_ui_clamp(n->scroll_offset_y,0,max_offset);if(max_offset>0){double bar=fmax(4,z_ui_prop_number(n->props,"scrollbarWidth",8)),gutter=fmax(2,z_ui_prop_number(n->props,"scrollbarGutter",4));inner.width=fmax(0,inner.width-bar-gutter);}}
    n->content_bounds=inner;n->content_height=content_height;
    double cursor_x=inner.x,cursor_y=inner.y-(scrollable?n->scroll_offset_y:0);const char*justify=z_ui_prop_string(n->props,"justify","start");
    if(grow_total==0&&remaining>0){if(strcmp(justify,"center")==0){if(row)cursor_x+=remaining/2;else cursor_y+=remaining/2;}else if(strcmp(justify,"end")==0){if(row)cursor_x+=remaining;else cursor_y+=remaining;}else if(strcmp(justify,"space-between")==0&&count>1)gap+=remaining/(double)(count-1);}
    for(size_t i=0;i<count;i++){ZUINodeNative*c=z_ui_expect_node(n->children.as.array->items[i]);double grow=z_ui_prop_number(c->props,"grow",0),cw,ch;if(row){cw=grow>0&&grow_total>0?remaining*grow/grow_total:z_ui_prop_number(c->props,"width",z_ui_natural_width(c,t));ch=inner.height;const char*align=z_ui_prop_string(n->props,"align","stretch");double natural=z_ui_natural_height(c,t);if(strcmp(align,"stretch")!=0){ch=z_ui_prop_number(c->props,"height",natural);if(ch>inner.height)ch=inner.height;}double cy=cursor_y;if(strcmp(align,"center")==0)cy=inner.y+(inner.height-ch)/2;else if(strcmp(align,"end")==0)cy=inner.y+inner.height-ch;z_ui_layout(c,(ZUIRectNative){cursor_x,cy,cw,ch},t);cursor_x+=cw+gap;}else{ch=grow>0&&grow_total>0?remaining*grow/grow_total:z_ui_prop_number(c->props,"height",z_ui_natural_height(c,t));cw=inner.width;const char*align=z_ui_prop_string(n->props,"align","stretch");double natural=z_ui_natural_width(c,t);if(strcmp(align,"stretch")!=0){cw=z_ui_prop_number(c->props,"width",natural);if(cw>inner.width)cw=inner.width;}double cx=cursor_x;if(strcmp(align,"center")==0)cx=inner.x+(inner.width-cw)/2;else if(strcmp(align,"end")==0)cx=inner.x+inner.width-cw;z_ui_layout(c,(ZUIRectNative){cx,cursor_y,cw,ch},t);cursor_y+=ch+gap;}}
}

static void z_ui_color_parse(const char*s,uint8_t*r,uint8_t*g,uint8_t*b,uint8_t*a){*r=*g=*b=0;*a=255;if(s==NULL||strcmp(s,"transparent")==0){*a=0;return;}if(*s=='#')s++;size_t n=strlen(s);unsigned long v=strtoul(s,NULL,16);if(n==6){*r=(v>>16)&255;*g=(v>>8)&255;*b=v&255;}else if(n==8){*r=(v>>24)&255;*g=(v>>16)&255;*b=(v>>8)&255;*a=v&255;}}
#if defined(ZUMBRA_DESKTOP_SDL3)
static void z_ui_draw_color(SDL_Renderer*r,const char*color){uint8_t red,green,blue,alpha;z_ui_color_parse(color,&red,&green,&blue,&alpha);z_desktop_sdl.SetRenderDrawColor(r,red,green,blue,alpha);}
static void z_ui_fill(SDL_Renderer*r,ZUIRectNative q,const char*color){uint8_t a,b,c,d;z_ui_color_parse(color,&a,&b,&c,&d);if(d==0)return;z_desktop_sdl.SetRenderDrawColor(r,a,b,c,d);ZDesktopSDLFRect f={(float)q.x,(float)q.y,(float)q.width,(float)q.height};z_desktop_sdl.RenderFillRect(r,&f);}
static void z_ui_stroke(SDL_Renderer*r,ZUIRectNative q,const char*color){z_ui_draw_color(r,color);ZDesktopSDLFRect f={(float)q.x,(float)q.y,(float)q.width,(float)q.height};z_desktop_sdl.RenderRect(r,&f);}
static void z_ui_text(SDL_Renderer*r,double x,double y,const char*text,const char*color,ZValue props,ZUIThemeNative*theme){
    if(text==NULL||*text=='\0')return;
    double size=z_ui_prop_number(props,"fontSize",z_ui_theme_number(theme,"fontSize",14));
    const char*family=z_ui_prop_string(props,"fontFamily",z_ui_theme_string(theme,"fontFamily","sans"));
    const char*path=z_ui_prop_string(props,"fontPath",z_ui_theme_string(theme,"fontPath",""));
    const char*weight=z_ui_prop_string(props,"fontWeight",z_ui_theme_string(theme,"fontWeight","normal"));
    const char*style=z_ui_prop_string(props,"fontStyle",z_ui_theme_string(theme,"fontStyle","normal"));
    double wrap=z_ui_prop_number(props,"wrapWidth",0);
    uint8_t red,green,blue,alpha;z_ui_color_parse(color,&red,&green,&blue,&alpha);
    TTF_Font*font=z_ui_font(family,path,size,weight,style);
    if(font&&z_desktop_sdl.CreateTextureFromSurface&&z_desktop_sdl.RenderTexture&&z_desktop_sdl.DestroyTexture&&z_desktop_sdl.DestroySurface){
        ZUITTFColor c={red,green,blue,alpha};
        SDL_Surface*surface=(wrap>0&&z_ui_ttf.RenderTextBlendedWrapped)?z_ui_ttf.RenderTextBlendedWrapped(font,text,0,c,(int)wrap):z_ui_ttf.RenderTextBlended(font,text,0,c);
        if(surface){
            SDL_Texture*texture=z_desktop_sdl.CreateTextureFromSurface(r,surface);
            z_desktop_sdl.DestroySurface(surface);
            if(texture){
                float width=0,height=0;
                if(z_desktop_sdl.GetTextureSize)z_desktop_sdl.GetTextureSize(texture,&width,&height);
                if(width<=0||height<=0){double mw=0,mh=0;z_ui_measure_text_native(text,props,theme,wrap,&mw,&mh);width=(float)mw;height=(float)mh;}
                ZDesktopSDLFRect dst={(float)x,(float)y,width,height};
                z_desktop_sdl.RenderTexture(r,texture,NULL,&dst);
                z_desktop_sdl.DestroyTexture(texture);
                return;
            }
        }
    }
    if(z_desktop_sdl.RenderDebugText){z_desktop_sdl.SetRenderDrawColor(r,red,green,blue,alpha);z_desktop_sdl.RenderDebugText(r,(float)x,(float)y,text);}
}
static void z_ui_text_centered(SDL_Renderer*r,ZUIRectNative bounds,double left_padding,const char*text,const char*color,ZValue props,ZUIThemeNative*theme){double width=0,height=0;z_ui_measure_text_native(text,props,theme,z_ui_prop_number(props,"wrapWidth",0),&width,&height);(void)width;double y=bounds.y+fmax(0,(bounds.height-height)/2);z_ui_text(r,bounds.x+left_padding,y,text,color,props,theme);}
static const char*z_ui_node_text(ZUINodeNative*n){return z_ui_props_text(n->props);}
static const char*z_ui_value_text(ZValue value){if(value.tag==ZV_STRING)return value.as.s;ZValue converted=z_to_string_value(value);return converted.tag==ZV_STRING?converted.as.s:"";}
static void z_ui_set_clip(SDL_Renderer*r,bool enabled,ZUIRectNative clip){if(z_desktop_sdl.SetRenderClipRect==NULL)return;if(!enabled){z_desktop_sdl.SetRenderClipRect(r,NULL);return;}ZDesktopSDLRect q={(int32_t)floor(clip.x),(int32_t)floor(clip.y),(int32_t)ceil(clip.width),(int32_t)ceil(clip.height)};z_desktop_sdl.SetRenderClipRect(r,&q);}
static void z_ui_render_node(SDL_Renderer*r,ZUINodeNative*n,ZUIContextNative*ctx,bool has_clip,ZUIRectNative clip){z_ui_set_clip(r,has_clip,clip);
    if(n==NULL||!n->visible)return;
    const char*surface=z_ui_theme_string(ctx->theme,"surface","#ffffff");
    const char*text=z_ui_theme_string(ctx->theme,"text","#172033");
    const char*border=z_ui_theme_string(ctx->theme,"border","#cfd6e2");
    const char*primary=z_ui_theme_string(ctx->theme,"primary","#3867e8");
    const char*bg=z_ui_prop_string(n->props,"background","transparent");
    const char*fg=z_ui_prop_string(n->props,"textColor",text);
    ZUIRectNative b=n->bounds;
    if(strcmp(n->kind,"text")==0){z_ui_text(r,b.x+2,b.y+4,z_ui_node_text(n),fg,n->props,ctx->theme);}
    else if(strcmp(n->kind,"button")==0){if(strcmp(bg,"transparent")==0)bg=primary;z_ui_fill(r,b,bg);z_ui_stroke(r,b,border);z_ui_text_centered(r,b,10,z_ui_node_text(n),z_ui_prop_string(n->props,"textColor",z_ui_theme_string(ctx->theme,"primaryText","#ffffff")),n->props,ctx->theme);}
    else if(strcmp(n->kind,"input")==0||strcmp(n->kind,"textarea")==0||strcmp(n->kind,"select")==0){if(strcmp(bg,"transparent")==0)bg=surface;z_ui_fill(r,b,bg);z_ui_stroke(r,b,border);const char*label=z_ui_node_text(n);if(*label=='\0')label=z_ui_prop_string(n->props,"placeholder","");z_ui_text_centered(r,b,8,label,fg,n->props,ctx->theme);if(strcmp(n->kind,"select")==0){ZUIRectNative arrow={b.x+b.width-24,b.y,20,b.height};z_ui_text_centered(r,arrow,2,"v",fg,n->props,ctx->theme);}}
    else if(strcmp(n->kind,"checkbox")==0||strcmp(n->kind,"radio")==0){ZUIRectNative box={b.x+4,b.y+(b.height-16)/2,16,16};z_ui_fill(r,box,surface);z_ui_stroke(r,box,border);if(z_ui_prop_bool(n->props,"checked",false))z_ui_fill(r,(ZUIRectNative){box.x+4,box.y+4,8,8},primary);ZUIRectNative label_bounds={b.x+20,b.y,b.width-20,b.height};z_ui_text_centered(r,label_bounds,8,z_ui_node_text(n),fg,n->props,ctx->theme);}
    else if(strcmp(n->kind,"progress")==0){z_ui_fill(r,b,z_ui_prop_string(n->props,"background",z_ui_theme_string(ctx->theme,"surfaceAlt","#eef2f7")));double max=z_ui_prop_number(n->props,"max",100),value=z_ui_prop_number(n->props,"value",0);if(max<=0)max=100;if(value<0)value=0;if(value>max)value=max;ZUIRectNative fill=b;fill.width=b.width*value/max;z_ui_fill(r,fill,z_ui_prop_string(n->props,"fill",primary));}
    else if(strcmp(n->kind,"list")==0||strcmp(n->kind,"tree")==0){z_ui_fill(r,b,bg);z_ui_stroke(r,b,border);ZValue items=z_desktop_dict_get(n->props,"items");double row_height=z_ui_prop_number(n->props,"itemHeight",28);if(items.tag==ZV_ARRAY){for(size_t i=0;i<items.as.array->len;i++){double y=b.y+6+(double)i*row_height;if(y>b.y+b.height)break;const char*label=z_ui_value_text(items.as.array->items[i]);if(strcmp(n->kind,"tree")==0){char*line=(char*)z_alloc(strlen(label)+3);sprintf(line,"- %s",label);z_ui_text(r,b.x+8,y,line,fg,n->props,ctx->theme);}else z_ui_text(r,b.x+8,y,label,fg,n->props,ctx->theme);}}}
    else if(strcmp(n->kind,"tabs")==0){z_ui_fill(r,b,bg);ZValue tabs=z_desktop_dict_get(n->props,"tabs");int64_t selected=(int64_t)z_ui_prop_number(n->props,"selectedIndex",0);double x=b.x;if(tabs.tag==ZV_ARRAY){for(size_t i=0;i<tabs.as.array->len;i++){const char*label=z_ui_value_text(tabs.as.array->items[i]);double tw=0,th=0;z_ui_measure_text_native(label,n->props,ctx->theme,0,&tw,&th);double width=fmax(72,tw+24);ZUIRectNative tab={x,b.y,width,b.height};if((int64_t)i==selected)z_ui_fill(r,tab,z_ui_prop_string(n->props,"selectedBackground",surface));z_ui_stroke(r,tab,border);z_ui_text_centered(r,tab,10,label,fg,n->props,ctx->theme);x+=width;}}}
    else if(strcmp(n->kind,"menu")==0){z_ui_fill(r,b,bg);z_ui_stroke(r,b,border);const char*label=z_ui_node_text(n);if(*label=='\0')label="Menu";z_ui_text_centered(r,b,10,label,fg,n->props,ctx->theme);}
    else if(strcmp(n->kind,"table")==0){z_ui_fill(r,b,bg);z_ui_stroke(r,b,border);ZValue columns=z_desktop_dict_get(n->props,"columns"),rows=z_desktop_dict_get(n->props,"rows");double row_height=z_ui_prop_number(n->props,"rowHeight",30);size_t count=columns.tag==ZV_ARRAY?columns.as.array->len:1;if(count<1)count=1;double column_width=b.width/(double)count;if(columns.tag==ZV_ARRAY){for(size_t i=0;i<columns.as.array->len;i++){double x=b.x+(double)i*column_width;z_ui_text(r,x+6,b.y+6,z_ui_value_text(columns.as.array->items[i]),fg,n->props,ctx->theme);if(i>0){z_ui_draw_color(r,border);z_desktop_sdl.RenderLine(r,(float)x,(float)b.y,(float)x,(float)(b.y+b.height));}}}z_ui_draw_color(r,border);z_desktop_sdl.RenderLine(r,(float)b.x,(float)(b.y+row_height),(float)(b.x+b.width),(float)(b.y+row_height));if(rows.tag==ZV_ARRAY){for(size_t i=0;i<rows.as.array->len;i++){double y=b.y+row_height+(double)i*row_height;if(y>b.y+b.height)break;z_ui_text(r,b.x+6,y+6,z_ui_value_text(rows.as.array->items[i]),fg,n->props,ctx->theme);z_ui_draw_color(r,border);z_desktop_sdl.RenderLine(r,(float)b.x,(float)(y+row_height),(float)(b.x+b.width),(float)(y+row_height));}}}
    else if(strcmp(n->kind,"image")==0){const char*path=z_ui_prop_string(n->props,"path","");if(*path&&z_desktop_sdl.IOFromFile&&z_desktop_sdl.LoadBMP_IO&&z_desktop_sdl.CreateTextureFromSurface&&z_desktop_sdl.RenderTexture&&z_desktop_sdl.DestroyTexture){SDL_Surface*s=z_desktop_sdl_load_bmp(path);if(s){SDL_Texture*texture=z_desktop_sdl.CreateTextureFromSurface(r,s);z_desktop_sdl.DestroySurface(s);if(texture){ZDesktopSDLFRect d={(float)b.x,(float)b.y,(float)b.width,(float)b.height};z_desktop_sdl.RenderTexture(r,texture,NULL,&d);z_desktop_sdl.DestroyTexture(texture);}}}}
    else if(strcmp(n->kind,"modal")==0){ZUIRectNative viewport={0,0,b.x*2+b.width,b.y*2+b.height};z_ui_fill(r,viewport,z_ui_prop_string(n->props,"overlay","#00000080"));z_ui_fill(r,b,strcmp(bg,"transparent")==0?surface:bg);z_ui_stroke(r,b,border);}
    else if(strcmp(n->kind,"tooltip")==0){z_ui_fill(r,b,strcmp(bg,"transparent")==0?"#202633":bg);z_ui_text(r,b.x+6,b.y+5,z_ui_node_text(n),z_ui_prop_string(n->props,"textColor","#ffffff"),n->props,ctx->theme);}
    else if(strcmp(n->kind,"canvas")==0){z_ui_fill(r,b,bg);ZValue commands=z_desktop_dict_get(n->props,"commands");if(commands.tag==ZV_ARRAY){for(size_t i=0;i<commands.as.array->len;i++){ZValue command=commands.as.array->items[i];const char*kind=z_ui_prop_string(command,"kind","");ZValue values=z_desktop_dict_get(command,"values");const char*color=z_ui_prop_string(values,"color",text);double x=b.x+z_ui_prop_number(values,"x",0),y=b.y+z_ui_prop_number(values,"y",0);if(strcmp(kind,"fillRect")==0)z_ui_fill(r,(ZUIRectNative){x,y,z_ui_prop_number(values,"width",10),z_ui_prop_number(values,"height",10)},color);else if(strcmp(kind,"rect")==0)z_ui_stroke(r,(ZUIRectNative){x,y,z_ui_prop_number(values,"width",10),z_ui_prop_number(values,"height",10)},color);else if(strcmp(kind,"line")==0){z_ui_draw_color(r,color);z_desktop_sdl.RenderLine(r,(float)x,(float)y,(float)(b.x+z_ui_prop_number(values,"x2",0)),(float)(b.y+z_ui_prop_number(values,"y2",0)));}else if(strcmp(kind,"text")==0)z_ui_text(r,x,y,z_ui_prop_string(values,"text",""),color,values,ctx->theme);}}}
    else{z_ui_fill(r,b,bg);if(z_ui_prop_bool(n->props,"border",false))z_ui_stroke(r,b,border);}
    if(z_ui_scrollable_y(n)&&n->content_height>n->content_bounds.height+0.5&&n->content_bounds.height>0){double bar=fmax(4,z_ui_prop_number(n->props,"scrollbarWidth",8)),gutter=fmax(2,z_ui_prop_number(n->props,"scrollbarGutter",4));ZUIRectNative track={n->content_bounds.x+n->content_bounds.width+gutter,n->content_bounds.y,bar,n->content_bounds.height};z_ui_fill(r,track,z_ui_prop_string(n->props,"scrollbarTrack","#e1e6ef"));double thumb_h=fmax(24,track.height*n->content_bounds.height/n->content_height);if(thumb_h>track.height)thumb_h=track.height;double max_offset=n->content_height-n->content_bounds.height,thumb_y=track.y;if(max_offset>0)thumb_y+=(track.height-thumb_h)*n->scroll_offset_y/max_offset;z_ui_fill(r,(ZUIRectNative){track.x,thumb_y,track.width,thumb_h},z_ui_prop_string(n->props,"scrollbarThumb","#9aa7ba"));}
    if(ctx->focus_id&&strcmp(ctx->focus_id,n->id)==0)z_ui_stroke(r,(ZUIRectNative){b.x-2,b.y-2,b.width+4,b.height+4},z_ui_theme_string(ctx->theme,"focus","#6e95ff"));
    bool child_has_clip=has_clip;ZUIRectNative child_clip=clip;if(z_ui_scrollable_y(n)){if(has_clip){if(!z_ui_rect_intersection(clip,n->content_bounds,&child_clip))return;}else child_clip=n->content_bounds;child_has_clip=true;}
    for(size_t i=0;i<n->children.as.array->len;i++)z_ui_render_node(r,z_ui_expect_node(n->children.as.array->items[i]),ctx,child_has_clip,child_clip);
    z_ui_set_clip(r,has_clip,clip);
}
#endif
static void z_ui_render_context(ZUIContextNative*ctx){if(ctx==NULL||ctx->closed||ctx->window==NULL||!ctx->window->open)return;z_ui_layout(ctx->root,(ZUIRectNative){0,0,(double)ctx->window->width,(double)ctx->window->height},ctx->theme);
#if defined(ZUMBRA_DESKTOP_SDL3)
    if(!ctx->app->headless){if(ctx->window->renderer==NULL){ctx->window->renderer=z_desktop_sdl.CreateRenderer((SDL_Window*)ctx->window->handle,NULL);if(ctx->window->renderer==NULL)z_fatal("uiRender could not create SDL3 renderer: %s",z_desktop_sdl.GetError());}SDL_Renderer*r=(SDL_Renderer*)ctx->window->renderer;z_ui_draw_color(r,z_ui_theme_string(ctx->theme,"background","#f5f7fb"));z_desktop_sdl.RenderClear(r);z_ui_render_node(r,ctx->root,ctx,false,(ZUIRectNative){0,0,0,0});z_ui_set_clip(r,false,(ZUIRectNative){0,0,0,0});z_desktop_sdl.RenderPresent(r);}
#endif
    ctx->dirty=false;}
static void z_ui_render_all(ZDesktopAppNative*app){if(app==NULL)return;for(ZUIContextNative*c=app->ui_contexts;c;c=c->next)if(c->dirty)z_ui_render_context(c);}
static ZUINodeNative*z_ui_hit_clipped(ZUINodeNative*n,double x,double y,bool has_clip,ZUIRectNative clip){if(n==NULL||!n->visible)return NULL;if(has_clip&&!z_ui_point_in_rect(x,y,clip))return NULL;bool child_has_clip=has_clip;ZUIRectNative child_clip=clip;if(z_ui_scrollable_y(n)){if(has_clip){if(!z_ui_rect_intersection(clip,n->content_bounds,&child_clip))return NULL;}else child_clip=n->content_bounds;child_has_clip=true;}for(size_t i=n->children.as.array->len;i>0;i--){ZUINodeNative*h=z_ui_hit_clipped(z_ui_expect_node(n->children.as.array->items[i-1]),x,y,child_has_clip,child_clip);if(h)return h;}return z_ui_point_in_rect(x,y,n->bounds)?n:NULL;}
static ZUINodeNative*z_ui_hit(ZUINodeNative*n,double x,double y){return z_ui_hit_clipped(n,x,y,false,(ZUIRectNative){0,0,0,0});}
static bool z_ui_focusable(ZUINodeNative*n){if(!n||!n->visible||!n->enabled)return false;const char*k=n->kind;return strcmp(k,"button")==0||strcmp(k,"input")==0||strcmp(k,"textarea")==0||strcmp(k,"select")==0||strcmp(k,"checkbox")==0||strcmp(k,"radio")==0||strcmp(k,"tabs")==0||strcmp(k,"menu")==0||z_ui_prop_bool(n->props,"focusable",false);}
static void z_ui_call_prop(ZUINodeNative*n,const char*name,ZValue event){ZValue h=z_desktop_dict_get(n->props,name);if(h.tag==ZV_FUNCTION||h.tag==ZV_BUILTIN||h.tag==ZV_BOUND_METHOD)(void)z_call(h,&event,1);}
static ZUINodeNative*z_ui_find_node(ZUINodeNative*n,const char*id){if(n==NULL)return NULL;if(strcmp(n->id,id)==0)return n;for(size_t i=0;i<n->children.as.array->len;i++){ZUINodeNative*r=z_ui_find_node(z_ui_expect_node(n->children.as.array->items[i]),id);if(r)return r;}return NULL;}
static ZValue z_ui_state_set_value(ZUIStateNative*s,ZValue value){s->value=value;for(ZUIBindingNative*b=s->bindings;b;b=b->next){z_ui_dict_set(b->node->props,b->property,value);if(b->node->context)b->node->context->dirty=true;}for(ZUISubscriberNative*h=s->subscribers;h;h=h->next)(void)z_call(h->handler,&value,1);return value;}
static ZUIStateNative*z_ui_bound_state(ZUINodeNative*n,const char*property){for(ZUIReverseBindingNative*b=n?n->bindings:NULL;b;b=b->next)if(strcmp(b->property,property)==0)return b->state;return NULL;}
static void z_ui_update_bound(ZUINodeNative*n,const char*property,ZValue value){ZUIStateNative*s=z_ui_bound_state(n,property);if(s)z_ui_state_set_value(s,value);}
static void z_ui_collect_focusable_nodes(ZUINodeNative*n,ZUINodeNative**items,size_t*count,size_t cap){if(!n||*count>=cap)return;if(z_ui_focusable(n))items[(*count)++]=n;for(size_t i=0;i<n->children.as.array->len;i++)z_ui_collect_focusable_nodes(z_ui_expect_node(n->children.as.array->items[i]),items,count,cap);}
static void z_ui_utf8_backspace(char*value){size_t n=strlen(value);if(n==0)return;size_t i=n-1;while(i>0&&(((unsigned char)value[i]&0xC0)==0x80))i--;value[i]='\0';}
static bool z_ui_accepts_text_input(ZUINodeNative*n){return n!=NULL&&n->visible&&n->enabled&&(strcmp(n->kind,"input")==0||strcmp(n->kind,"textarea")==0);}
static void z_ui_sync_text_input(ZUIContextNative*ctx,ZUINodeNative*n){
#if defined(ZUMBRA_DESKTOP_SDL3)
    if(ctx==NULL||ctx->app==NULL||ctx->window==NULL||ctx->window->handle==NULL||ctx->app->headless)return;
    SDL_Window*w=(SDL_Window*)ctx->window->handle;
    if(z_ui_accepts_text_input(n))z_desktop_sdl.StartTextInput(w);else z_desktop_sdl.StopTextInput(w);
#else
    (void)ctx;(void)n;
#endif
}
static void z_ui_dispatch_event(ZUIContextNative*ctx,ZDesktopEventNative*event){
    if(!ctx||ctx->closed||!event)return;
    double x=z_ui_prop_number(event->data,"x",0),y=z_ui_prop_number(event->data,"y",0);
    ZUINodeNative*target=NULL;
    bool pointer_event=strcmp(event->type,"mouse_motion")==0||strcmp(event->type,"mouse_move")==0||strcmp(event->type,"mouse_down")==0||strcmp(event->type,"mouse_button_down")==0||strcmp(event->type,"mouse_up")==0||strcmp(event->type,"mouse_button_up")==0||strcmp(event->type,"mouse_wheel")==0;
    if(pointer_event)target=z_ui_hit(ctx->root,x,y);
    else if(strcmp(event->type,"key_down")==0||strcmp(event->type,"text_input")==0)target=z_ui_find_node(ctx->root,ctx->focus_id?ctx->focus_id:"");
    if(strcmp(event->type,"mouse_wheel")==0){ZUINodeNative*scroll=target;while(scroll&&!z_ui_scrollable_y(scroll))scroll=scroll->parent;if(scroll){double dy=z_ui_prop_number(event->data,"dy",z_ui_prop_number(event->data,"wheelY",0)),step=z_ui_prop_number(scroll->props,"scrollStep",48),max_offset=fmax(0,scroll->content_height-scroll->content_bounds.height);scroll->scroll_offset_y=z_ui_clamp(scroll->scroll_offset_y-dy*step,0,max_offset);ctx->dirty=true;}return;}
    const char*shortcut=z_ui_prop_string(event->data,"shortcut","");
    if(strcmp(event->type,"key_down")==0&&(strcasecmp(shortcut,"tab")==0||strcasecmp(shortcut,"shift+tab")==0)){
        ZUINodeNative*nodes[2048];size_t count=0;z_ui_collect_focusable_nodes(ctx->root,nodes,&count,2048);
        if(count){size_t current=count;for(size_t i=0;i<count;i++)if(ctx->focus_id&&strcmp(nodes[i]->id,ctx->focus_id)==0){current=i;break;}bool reverse=strcasecmp(shortcut,"shift+tab")==0;size_t next;if(current==count)next=reverse?count-1:0;else next=reverse?(current+count-1)%count:(current+1)%count;ctx->focus_id=z_strdup(nodes[next]->id);z_ui_sync_text_input(ctx,nodes[next]);ctx->dirty=true;}
        return;
    }
    bool down=strcmp(event->type,"mouse_down")==0||strcmp(event->type,"mouse_button_down")==0;
    if(!target||!target->enabled){if(down){ctx->focus_id=NULL;z_ui_sync_text_input(ctx,NULL);ctx->dirty=true;}return;}
    ZValue ev=z_desktop_event_value(event);z_ui_dict_set(ev,"targetId",z_string(target->id));z_ui_dict_set(ev,"targetKind",z_string(target->kind));
    bool up=strcmp(event->type,"mouse_up")==0||strcmp(event->type,"mouse_button_up")==0;
    if(down){
        if(z_ui_focusable(target)){ctx->focus_id=z_strdup(target->id);z_ui_sync_text_input(ctx,target);z_ui_call_prop(target,"onFocus",ev);}
        else{ctx->focus_id=NULL;z_ui_sync_text_input(ctx,NULL);}
    }
    if(strcmp(event->type,"mouse_move")==0||strcmp(event->type,"mouse_motion")==0){ctx->hover_id=z_strdup(target->id);z_ui_call_prop(target,"onHover",ev);}
    const char*key=z_ui_prop_string(event->data,"key","");
    bool activate=up||(strcmp(event->type,"key_down")==0&&(strcasecmp(key,"enter")==0||strcasecmp(key,"space")==0));
    if(activate){if(strcmp(target->kind,"button")==0||strcmp(target->kind,"menu")==0)z_ui_call_prop(target,"onClick",ev);else if(strcmp(target->kind,"checkbox")==0){bool checked=!z_ui_prop_bool(target->props,"checked",false);ZValue value=z_bool(checked);z_ui_dict_set(target->props,"checked",value);z_ui_update_bound(target,"checked",value);z_ui_call_prop(target,"onChange",ev);}else if(strcmp(target->kind,"radio")==0){ZValue value=z_bool(true);z_ui_dict_set(target->props,"checked",value);z_ui_update_bound(target,"checked",value);if(target->parent){for(size_t i=0;i<target->parent->children.as.array->len;i++){ZUINodeNative*sibling=z_ui_expect_node(target->parent->children.as.array->items[i]);if(sibling!=target&&strcmp(sibling->kind,"radio")==0)z_ui_dict_set(sibling->props,"checked",z_bool(false));}}z_ui_call_prop(target,"onChange",ev);}else if(strcmp(target->kind,"select")==0||strcmp(target->kind,"tabs")==0){const char*property=strcmp(target->kind,"tabs")==0?"tabs":"options";ZValue items=z_desktop_dict_get(target->props,property);if(items.tag==ZV_ARRAY&&items.as.array->len>0){int64_t index=(int64_t)z_ui_prop_number(target->props,"selectedIndex",0);index=(index+1)%(int64_t)items.as.array->len;ZValue selected_index=z_int(index),selected_value=z_string(z_ui_value_text(items.as.array->items[index]));z_ui_dict_set(target->props,"selectedIndex",selected_index);z_ui_dict_set(target->props,"value",selected_value);z_ui_update_bound(target,"selectedIndex",selected_index);z_ui_update_bound(target,"value",selected_value);z_ui_call_prop(target,"onChange",ev);}}}
    if(strcmp(event->type,"key_down")==0&&(strcmp(target->kind,"input")==0||strcmp(target->kind,"textarea")==0)&&strcasecmp(key,"backspace")==0){const char*current=z_ui_prop_string(target->props,"value","");char*next=z_strdup(current);z_ui_utf8_backspace(next);ZValue value=z_string(next);z_ui_dict_set(target->props,"value",value);z_ui_update_bound(target,"value",value);z_ui_call_prop(target,"onInput",ev);z_ui_call_prop(target,"onChange",ev);}
    if(strcmp(event->type,"text_input")==0&&(strcmp(target->kind,"input")==0||strcmp(target->kind,"textarea")==0)){const char*current=z_ui_prop_string(target->props,"value",""),*added=z_ui_prop_string(event->data,"text","");size_t length=strlen(current)+strlen(added)+1;char*next=(char*)z_alloc(length);snprintf(next,length,"%s%s",current,added);ZValue value=z_string(next);z_ui_dict_set(target->props,"value",value);z_ui_update_bound(target,"value",value);z_ui_call_prop(target,"onInput",ev);z_ui_call_prop(target,"onChange",ev);}
    ctx->dirty=true;
}
static void z_ui_dispatch_from_desktop(ZDesktopAppNative*app,ZDesktopEventNative*event){if(!app||!event)return;for(ZUIContextNative*c=app->ui_contexts;c;c=c->next)if(c->window&&c->window->id==event->window_id)z_ui_dispatch_event(c,event);}
static ZValue z_ui_snapshot_node(ZUINodeNative*n){ZValue pairs[11]={z_pair(z_string("id"),z_string(n->id)),z_pair(z_string("kind"),z_string(n->kind)),z_pair(z_string("x"),z_float(n->bounds.x)),z_pair(z_string("y"),z_float(n->bounds.y)),z_pair(z_string("width"),z_float(n->bounds.width)),z_pair(z_string("height"),z_float(n->bounds.height)),z_pair(z_string("contentX"),z_float(n->content_bounds.x)),z_pair(z_string("contentY"),z_float(n->content_bounds.y)),z_pair(z_string("contentWidth"),z_float(n->content_bounds.width)),z_pair(z_string("contentHeight"),z_float(n->content_height)),z_pair(z_string("scrollOffsetY"),z_float(n->scroll_offset_y))};return z_dict_from(pairs,11);}
static void z_ui_collect_snapshot(ZUINodeNative*n,ZValue*items,size_t*count,size_t cap){if(!n||*count>=cap)return;items[(*count)++]=z_ui_snapshot_node(n);for(size_t i=0;i<n->children.as.array->len;i++)z_ui_collect_snapshot(z_ui_expect_node(n->children.as.array->items[i]),items,count,cap);}
static ZValue z_ui_snapshot(ZUIContextNative*ctx){z_ui_render_context(ctx);ZValue items[2048];size_t count=0;z_ui_collect_snapshot(ctx->root,items,&count,2048);ZValue pairs[4]={z_pair(z_string("width"),z_float(ctx->window->width)),z_pair(z_string("height"),z_float(ctx->window->height)),z_pair(z_string("background"),z_string(z_ui_theme_string(ctx->theme,"background","#f5f7fb"))),z_pair(z_string("items"),z_array_from(items,count))};return z_dict_from(pairs,4);}
static void z_ui_collect_accessibility(ZUINodeNative*n,ZValue*items,size_t*count,size_t cap){if(!n||*count>=cap)return;const char*label=z_ui_prop_string(n->props,"accessibilityLabel",z_ui_prop_string(n->props,"text",z_ui_prop_string(n->props,"label","")));ZValue pairs[5]={z_pair(z_string("id"),z_string(n->id)),z_pair(z_string("role"),z_string(z_ui_prop_string(n->props,"role",n->kind))),z_pair(z_string("label"),z_string(label)),z_pair(z_string("enabled"),z_bool(n->enabled)),z_pair(z_string("focusable"),z_bool(z_ui_focusable(n)))};items[(*count)++]=z_dict_from(pairs,5);for(size_t i=0;i<n->children.as.array->len;i++)z_ui_collect_accessibility(z_ui_expect_node(n->children.as.array->items[i]),items,count,cap);}
static ZValue z_ui_method(ZValue receiver,const char*builtin){ZUIMethodNative*m=(ZUIMethodNative*)z_alloc(sizeof(*m));m->receiver=receiver;m->builtin=builtin;return z_ui_value(ZV_UI_METHOD,ZK_UNKNOWN,m);}
static ZValue z_ui_call_method(ZValue callable,const ZValue*args,size_t argc){ZUIMethodNative*m=(ZUIMethodNative*)callable.as.p;ZValue*full=(ZValue*)z_alloc(sizeof(ZValue)*(argc+1));full[0]=m->receiver;if(argc)memcpy(full+1,args,sizeof(ZValue)*argc);return z_ui_call_builtin(m->builtin,full,argc+1,NULL);}
static ZValue z_ui_get_field(ZValue value,const char*name,bool*handled){*handled=true;if(value.tag==ZV_UI_CONTEXT){if(strcmp(name,"render")==0)return z_ui_method(value,"uiRender");if(strcmp(name,"snapshot")==0)return z_ui_method(value,"uiSnapshot");if(strcmp(name,"setTheme")==0)return z_ui_method(value,"uiSetTheme");if(strcmp(name,"dispatch")==0)return z_ui_method(value,"uiDispatch");if(strcmp(name,"find")==0)return z_ui_method(value,"uiFind");if(strcmp(name,"focus")==0)return z_ui_method(value,"uiFocus");if(strcmp(name,"focusNext")==0)return z_ui_method(value,"uiFocusNext");if(strcmp(name,"accessibility")==0)return z_ui_method(value,"uiAccessibility");if(strcmp(name,"close")==0)return z_ui_method(value,"uiUnmount");}if(value.tag==ZV_UI_NODE){if(strcmp(name,"set")==0)return z_ui_method(value,"uiSet");if(strcmp(name,"get")==0)return z_ui_method(value,"uiGet");if(strcmp(name,"add")==0)return z_ui_method(value,"uiAdd");if(strcmp(name,"remove")==0)return z_ui_method(value,"uiRemove");if(strcmp(name,"command")==0)return z_ui_method(value,"uiCanvasCommand");}if(value.tag==ZV_UI_STATE){if(strcmp(name,"get")==0)return z_ui_method(value,"uiStateGet");if(strcmp(name,"set")==0)return z_ui_method(value,"uiStateSet");if(strcmp(name,"subscribe")==0)return z_ui_method(value,"uiStateSubscribe");}*handled=false;return z_null();}
static ZValue z_ui_call_builtin(const char*name,const ZValue*args,size_t argc,bool*handled){if(handled)*handled=true;
    if(strcmp(name,"uiTheme")==0){z_expect_args(name,argc,1);return z_ui_value(ZV_UI_THEME,ZK_UI_THEME,z_ui_theme_new(args[0]));}
    if(strcmp(name,"uiState")==0){z_expect_args(name,argc,1);ZUIStateNative*s=(ZUIStateNative*)z_alloc(sizeof(*s));memset(s,0,sizeof(*s));s->value=args[0];return z_ui_value(ZV_UI_STATE,ZK_UI_STATE,s);}
    if(strcmp(name,"uiStateGet")==0){z_expect_args(name,argc,1);return z_ui_expect_state(args[0])->value;}
    if(strcmp(name,"uiStateSet")==0){z_expect_args(name,argc,2);return z_ui_state_set_value(z_ui_expect_state(args[0]),args[1]);}
    if(strcmp(name,"uiStateSubscribe")==0){z_expect_args(name,argc,2);ZUIStateNative*s=z_ui_expect_state(args[0]);ZUISubscriberNative*h=(ZUISubscriberNative*)z_alloc(sizeof(*h));h->handler=args[1];h->next=s->subscribers;s->subscribers=h;return args[0];}
    if(strcmp(name,"uiBind")==0){z_expect_args(name,argc,3);ZUINodeNative*n=z_ui_expect_node(args[0]);ZUIStateNative*s=z_ui_expect_state(args[2]);const char*property=z_as_cstring(args[1]);ZUIBindingNative*b=(ZUIBindingNative*)z_alloc(sizeof(*b));b->node=n;b->property=z_strdup(property);b->next=s->bindings;s->bindings=b;ZUIReverseBindingNative*r=(ZUIReverseBindingNative*)z_alloc(sizeof(*r));r->property=z_strdup(property);r->state=s;r->next=n->bindings;n->bindings=r;z_ui_dict_set(n->props,property,s->value);return args[0];}
    const char*kind=NULL;if(strcmp(name,"uiNode")==0){z_expect_args(name,argc,3);kind=z_as_cstring(args[0]);ZUINodeNative*n=z_ui_node_new(kind,args[1],args[2]);return z_ui_value(ZV_UI_NODE,ZK_UI_NODE,n);}const char*prefix="ui";if(strncmp(name,prefix,2)==0){const char*map_names[][2]={{"uiRow","row"},{"uiColumn","column"},{"uiContainer","container"},{"uiText","text"},{"uiButton","button"},{"uiInput","input"},{"uiTextarea","textarea"},{"uiSelect","select"},{"uiCheckbox","checkbox"},{"uiRadio","radio"},{"uiTable","table"},{"uiList","list"},{"uiTree","tree"},{"uiTabs","tabs"},{"uiMenu","menu"},{"uiModal","modal"},{"uiTooltip","tooltip"},{"uiProgress","progress"},{"uiImage","image"},{"uiCanvas","canvas"},{"uiSpacer","spacer"},{"uiCustom","custom"}};for(size_t i=0;i<sizeof(map_names)/sizeof(map_names[0]);i++)if(strcmp(name,map_names[i][0])==0){z_expect_args(name,argc,2);return z_ui_value(ZV_UI_NODE,ZK_UI_NODE,z_ui_node_new(map_names[i][1],args[0],args[1]));}}
    if(strcmp(name,"uiMount")==0){z_expect_args(name,argc,4);ZDesktopAppNative*app=z_desktop_expect_app(args[0]);ZDesktopWindowNative*w=z_desktop_expect_window(args[1]);ZUIContextNative*c=(ZUIContextNative*)z_alloc(sizeof(*c));memset(c,0,sizeof(*c));c->app=app;c->window=w;c->root=z_ui_expect_node(args[2]);ZValue theme=z_desktop_dict_get(args[3],"theme");c->theme=theme.tag==ZV_UI_THEME?z_ui_expect_theme(theme):z_ui_theme_new(z_string("light"));c->dirty=true;c->next=app->ui_contexts;app->ui_contexts=c;z_ui_attach_tree(c,c->root);z_ui_render_context(c);return z_ui_value(ZV_UI_CONTEXT,ZK_UI_CONTEXT,c);}
    if(strcmp(name,"uiUnmount")==0){z_expect_args(name,argc,1);ZUIContextNative*c=z_ui_expect_context(args[0]);bool changed=!c->closed;z_ui_sync_text_input(c,NULL);c->focus_id=NULL;c->closed=true;return z_bool(changed);}
    if(strcmp(name,"uiRender")==0){z_expect_args(name,argc,1);z_ui_render_context(z_ui_expect_context(args[0]));return z_bool(true);}
    if(strcmp(name,"uiSnapshot")==0){z_expect_args(name,argc,1);return z_ui_snapshot(z_ui_expect_context(args[0]));}
    if(strcmp(name,"uiSetTheme")==0){z_expect_args(name,argc,2);ZUIContextNative*c=z_ui_expect_context(args[0]);c->theme=z_ui_expect_theme(args[1]);c->dirty=true;z_ui_render_context(c);return args[0];}
    if(strcmp(name,"uiDispatch")==0){z_expect_args(name,argc,2);ZUIContextNative*c=z_ui_expect_context(args[0]);if(args[1].tag!=ZV_DICT)z_fatal("uiDispatch event must be dictionary");ZDesktopEventNative e={0};e.type=z_strdup(z_ui_prop_string(args[1],"type","custom"));e.window_id=c->window->id;e.timestamp=z_desktop_now_ns();e.data=args[1];z_ui_dispatch_event(c,&e);z_ui_render_context(c);return z_bool(true);}
    if(strcmp(name,"uiSet")==0){z_expect_args(name,argc,3);ZUINodeNative*n=z_ui_expect_node(args[0]);const char*property=z_as_cstring(args[1]);z_ui_dict_set(n->props,property,args[2]);if(strcmp(property,"visible")==0&&args[2].tag==ZV_BOOL)n->visible=args[2].as.b;if(strcmp(property,"disabled")==0&&args[2].tag==ZV_BOOL)n->enabled=!args[2].as.b;if(n->context)n->context->dirty=true;return args[0];}
    if(strcmp(name,"uiGet")==0){z_expect_args(name,argc,2);ZUINodeNative*n=z_ui_expect_node(args[0]);const char*property=z_as_cstring(args[1]);if(strcmp(property,"id")==0)return z_string(n->id);if(strcmp(property,"kind")==0)return z_string(n->kind);return z_desktop_dict_get(n->props,property);}
    if(strcmp(name,"uiAdd")==0){z_expect_args(name,argc,2);ZUINodeNative*p=z_ui_expect_node(args[0]),*child=z_ui_expect_node(args[1]);size_t len=p->children.as.array->len;ZValue*items=(ZValue*)z_alloc(sizeof(ZValue)*(len+1));if(len)memcpy(items,p->children.as.array->items,sizeof(ZValue)*len);items[len]=args[1];p->children=z_array_from(items,len+1);child->parent=p;if(p->context)z_ui_attach_tree(p->context,child);if(p->context)p->context->dirty=true;return args[0];}
    if(strcmp(name,"uiRemove")==0){z_expect_args(name,argc,2);ZUINodeNative*p=z_ui_expect_node(args[0]);const char*id=z_as_cstring(args[1]);size_t len=p->children.as.array->len;ZValue*items=(ZValue*)z_alloc(sizeof(ZValue)*len);size_t count=0;bool removed=false;for(size_t i=0;i<len;i++){ZUINodeNative*c=z_ui_expect_node(p->children.as.array->items[i]);if(strcmp(c->id,id)==0){removed=true;c->parent=NULL;}else items[count++]=p->children.as.array->items[i];}p->children=z_array_from(items,count);if(p->context)p->context->dirty=true;return z_bool(removed);}
    if(strcmp(name,"uiFind")==0){z_expect_args(name,argc,2);ZUINodeNative*n=z_ui_find_node(z_ui_expect_context(args[0])->root,z_as_cstring(args[1]));return n?z_ui_value(ZV_UI_NODE,ZK_UI_NODE,n):z_null();}
    if(strcmp(name,"uiFocus")==0){z_expect_args(name,argc,2);ZUIContextNative*c=z_ui_expect_context(args[0]);ZUINodeNative*n=z_ui_expect_node(args[1]);if(!z_ui_focusable(n))return z_bool(false);c->focus_id=z_strdup(n->id);z_ui_sync_text_input(c,n);c->dirty=true;return z_bool(true);}
    if(strcmp(name,"uiFocusNext")==0){z_expect_args(name,argc,2);ZUIContextNative*c=z_ui_expect_context(args[0]);bool reverse=args[1].tag==ZV_BOOL&&args[1].as.b;ZUINodeNative*nodes[2048];size_t count=0;z_ui_collect_focusable_nodes(c->root,nodes,&count,2048);if(count==0)return z_null();size_t current=count;for(size_t i=0;i<count;i++)if(c->focus_id&&strcmp(nodes[i]->id,c->focus_id)==0){current=i;break;}size_t next=current==count?(reverse?count-1:0):(reverse?(current+count-1)%count:(current+1)%count);c->focus_id=z_strdup(nodes[next]->id);z_ui_sync_text_input(c,nodes[next]);c->dirty=true;return z_ui_value(ZV_UI_NODE,ZK_UI_NODE,nodes[next]);}
    if(strcmp(name,"uiAccessibility")==0){z_expect_args(name,argc,1);ZValue items[2048];size_t count=0;z_ui_collect_accessibility(z_ui_expect_context(args[0])->root,items,&count,2048);return z_array_from(items,count);}
    if(strcmp(name,"uiCanvasCommand")==0){z_expect_args(name,argc,3);ZUINodeNative*n=z_ui_expect_node(args[0]);if(strcmp(n->kind,"canvas")!=0)z_fatal("uiCanvasCommand expects canvas");ZValue commands=z_desktop_dict_get(n->props,"commands");size_t len=commands.tag==ZV_ARRAY?commands.as.array->len:0;ZValue*items=(ZValue*)z_alloc(sizeof(ZValue)*(len+1));if(len)memcpy(items,commands.as.array->items,sizeof(ZValue)*len);ZValue pairs[2]={z_pair(z_string("kind"),args[1]),z_pair(z_string("values"),args[2])};items[len]=z_dict_from(pairs,2);z_ui_dict_set(n->props,"commands",z_array_from(items,len+1));if(n->context)n->context->dirty=true;return args[0];}
    if(handled) { *handled=false; }
    return z_null();
}

#endif
