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
static void z_ui_runtime_init(void){z_ui_next_id=1;}
static void z_ui_runtime_shutdown(void){}
static ZValue z_ui_value(ZTag tag,ZKind kind,void*p){ZValue v=z_value(tag,kind);v.as.p=p;return v;}
static ZUINodeNative*z_ui_expect_node(ZValue v){if(v.tag!=ZV_UI_NODE||v.as.p==NULL)z_fatal("expected UINode");return(ZUINodeNative*)v.as.p;}
static ZUIStateNative*z_ui_expect_state(ZValue v){if(v.tag!=ZV_UI_STATE||v.as.p==NULL)z_fatal("expected UIState");return(ZUIStateNative*)v.as.p;}
static ZUIThemeNative*z_ui_expect_theme(ZValue v){if(v.tag!=ZV_UI_THEME||v.as.p==NULL)z_fatal("expected UITheme");return(ZUIThemeNative*)v.as.p;}
static ZUIContextNative*z_ui_expect_context(ZValue v){if(v.tag!=ZV_UI_CONTEXT||v.as.p==NULL)z_fatal("expected UIContext");return(ZUIContextNative*)v.as.p;}
static double z_ui_number(ZValue v,double fallback){if(v.tag==ZV_FLOAT)return v.as.f;if(v.tag==ZV_INT)return(double)v.as.i;if(v.tag==ZV_UINT)return(double)v.as.u;return fallback;}
static double z_ui_prop_number(ZValue props,const char*key,double fallback){return z_ui_number(z_desktop_dict_get(props,key),fallback);}
static const char*z_ui_prop_string(ZValue props,const char*key,const char*fallback){return z_desktop_opt_string(props,key,fallback);}
static bool z_ui_prop_bool(ZValue props,const char*key,bool fallback){return z_desktop_opt_bool(props,key,fallback);}
static void z_ui_dict_set(ZValue dict,const char*key,ZValue value){z_set_index(dict,z_string(key),value);}
static ZValue z_ui_theme_values(bool dark){
    ZValue pairs[13]={
        z_pair(z_string("background"),z_string(dark?"#141822":"#f5f7fb")),z_pair(z_string("surface"),z_string(dark?"#202633":"#ffffff")),
        z_pair(z_string("surfaceAlt"),z_string(dark?"#2a3242":"#eef2f7")),z_pair(z_string("text"),z_string(dark?"#f2f5fa":"#172033")),
        z_pair(z_string("muted"),z_string(dark?"#a9b4c7":"#697386")),z_pair(z_string("primary"),z_string("#3867e8")),
        z_pair(z_string("primaryText"),z_string("#ffffff")),z_pair(z_string("border"),z_string(dark?"#465066":"#cfd6e2")),
        z_pair(z_string("danger"),z_string("#c73737")),z_pair(z_string("focus"),z_string("#6e95ff")),
        z_pair(z_string("radius"),z_int(6)),z_pair(z_string("fontSize"),z_int(14)),z_pair(z_string("spacing"),z_int(8))};
    ZValue result=z_dict_from(pairs,13);z_ui_dict_set(result,"controlHeight",z_int(36));return result;
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
static double z_ui_natural_height(ZUINodeNative*n,ZUIThemeNative*t){
    double control=z_ui_theme_number(t,"controlHeight",36),font=z_ui_prop_number(n->props,"fontSize",z_ui_theme_number(t,"fontSize",14));
    if(strcmp(n->kind,"text")==0)return font+8;
    if(strcmp(n->kind,"textarea")==0)return z_ui_prop_number(n->props,"rows",4)*(font+5)+16;
    if(strcmp(n->kind,"table")==0){size_t rows=z_ui_array_length(z_desktop_dict_get(n->props,"rows"));if(rows==0)rows=1;return(double)(rows+1)*control;}
    if(strcmp(n->kind,"list")==0||strcmp(n->kind,"tree")==0){size_t items=z_ui_array_length(z_desktop_dict_get(n->props,"items"));if(items==0)items=1;return(double)items*control;}
    if(strcmp(n->kind,"tabs")==0||strcmp(n->kind,"menu")==0)return control;
    if(strcmp(n->kind,"modal")==0)return z_ui_prop_number(n->props,"contentHeight",320);
    if(strcmp(n->kind,"tooltip")==0)return font+16;
    if(strcmp(n->kind,"progress")==0)return 20;
    if(strcmp(n->kind,"image")==0||strcmp(n->kind,"canvas")==0)return z_ui_prop_number(n->props,"height",160);
    if(strcmp(n->kind,"spacer")==0)return z_ui_prop_number(n->props,"size",8);
    return z_ui_prop_number(n->props,"height",control);
}
static double z_ui_natural_width(ZUINodeNative*n,ZUIThemeNative*t){
    (void)t;const char*text=z_ui_prop_string(n->props,"text",z_ui_prop_string(n->props,"label",z_ui_prop_string(n->props,"value","")));double width=(double)strlen(text)*8+24;if(width<48)width=48;
    if(strcmp(n->kind,"input")==0||strcmp(n->kind,"textarea")==0||strcmp(n->kind,"select")==0)if(width<180)width=180;
    if(strcmp(n->kind,"checkbox")==0||strcmp(n->kind,"radio")==0)width+=20;
    if(strcmp(n->kind,"image")==0||strcmp(n->kind,"canvas")==0)width=z_ui_prop_number(n->props,"width",240);
    if(strcmp(n->kind,"spacer")==0) { width=z_ui_prop_number(n->props,"size",8); }
    return width;
}
static void z_ui_layout(ZUINodeNative*n,ZUIRectNative available,ZUIThemeNative*t){
    if(n==NULL||!n->visible) { return; }
    ZUIEdgesNative margin=z_ui_edges(n->props,"margin",0);
    double x=available.x+margin.left,y=available.y+margin.top,width=available.width-margin.left-margin.right,height=available.height-margin.top-margin.bottom;
    ZValue wv=z_desktop_dict_get(n->props,"width"),hv=z_desktop_dict_get(n->props,"height");if(wv.tag!=ZV_NULL)width=z_ui_number(wv,width);if(hv.tag!=ZV_NULL)height=z_ui_number(hv,height);
    double minw=z_ui_prop_number(n->props,"minWidth",width),maxw=z_ui_prop_number(n->props,"maxWidth",width),minh=z_ui_prop_number(n->props,"minHeight",height),maxh=z_ui_prop_number(n->props,"maxHeight",height);if(width<minw)width=minw;if(width>maxw)width=maxw;if(height<minh)height=minh;if(height>maxh)height=maxh;if(width<0)width=0;if(height<0)height=0;
    if(hv.tag==ZV_NULL&&strcmp(n->kind,"row")!=0&&strcmp(n->kind,"column")!=0&&strcmp(n->kind,"container")!=0&&strcmp(n->kind,"modal")!=0){double natural=z_ui_natural_height(n,t);if(height>natural)height=natural;}
    if(strcmp(n->kind,"modal")==0){double mw=z_ui_prop_number(n->props,"contentWidth",480),mh=z_ui_prop_number(n->props,"contentHeight",320);if(mw>width)mw=width;if(mh>height)mh=height;x+=(width-mw)/2;y+=(height-mh)/2;width=mw;height=mh;}
    n->bounds=(ZUIRectNative){x,y,width,height};size_t count=n->children.as.array->len;if(count==0)return;
    ZUIEdgesNative padding=z_ui_edges(n->props,"padding",z_ui_theme_number(t,"spacing",8));double gap=z_ui_prop_number(n->props,"gap",z_ui_theme_number(t,"spacing",8));if(gap<0)gap=0;ZUIRectNative inner={x+padding.left,y+padding.top,width-padding.left-padding.right,height-padding.top-padding.bottom};if(inner.width<0)inner.width=0;if(inner.height<0)inner.height=0;
    const char*direction=n->kind;if(strcmp(direction,"row")!=0&&strcmp(direction,"column")!=0)direction=z_ui_prop_string(n->props,"direction","column");bool row=strcmp(direction,"row")==0;double main=(row?inner.width:inner.height)-gap*(double)(count>0?count-1:0);if(main<0)main=0;double fixed=0,grow_total=0;
    for(size_t i=0;i<count;i++){ZUINodeNative*c=z_ui_expect_node(n->children.as.array->items[i]);double grow=z_ui_prop_number(c->props,"grow",0);if(grow>0)grow_total+=grow;else fixed+=row?z_ui_prop_number(c->props,"width",z_ui_natural_width(c,t)):z_ui_prop_number(c->props,"height",z_ui_natural_height(c,t));}
    double remaining=main-fixed;if(remaining<0)remaining=0;double cursor_x=inner.x,cursor_y=inner.y;const char*justify=z_ui_prop_string(n->props,"justify","start");if(grow_total==0&&remaining>0){if(strcmp(justify,"center")==0){if(row)cursor_x+=remaining/2;else cursor_y+=remaining/2;}else if(strcmp(justify,"end")==0){if(row)cursor_x+=remaining;else cursor_y+=remaining;}else if(strcmp(justify,"space-between")==0&&count>1)gap+=remaining/(double)(count-1);}
    for(size_t i=0;i<count;i++){ZUINodeNative*c=z_ui_expect_node(n->children.as.array->items[i]);double grow=z_ui_prop_number(c->props,"grow",0),cw,ch;if(row){cw=grow>0&&grow_total>0?remaining*grow/grow_total:z_ui_prop_number(c->props,"width",z_ui_natural_width(c,t));ch=inner.height;const char*align=z_ui_prop_string(n->props,"align","stretch");double natural=z_ui_natural_height(c,t);if(strcmp(align,"stretch")!=0){ch=z_ui_prop_number(c->props,"height",natural);if(ch>inner.height)ch=inner.height;}double cy=cursor_y;if(strcmp(align,"center")==0)cy=inner.y+(inner.height-ch)/2;else if(strcmp(align,"end")==0)cy=inner.y+inner.height-ch;z_ui_layout(c,(ZUIRectNative){cursor_x,cy,cw,ch},t);cursor_x+=cw+gap;}else{ch=grow>0&&grow_total>0?remaining*grow/grow_total:z_ui_prop_number(c->props,"height",z_ui_natural_height(c,t));cw=inner.width;const char*align=z_ui_prop_string(n->props,"align","stretch");double natural=z_ui_natural_width(c,t);if(strcmp(align,"stretch")!=0){cw=z_ui_prop_number(c->props,"width",natural);if(cw>inner.width)cw=inner.width;}double cx=cursor_x;if(strcmp(align,"center")==0)cx=inner.x+(inner.width-cw)/2;else if(strcmp(align,"end")==0)cx=inner.x+inner.width-cw;z_ui_layout(c,(ZUIRectNative){cx,cursor_y,cw,ch},t);cursor_y+=ch+gap;}}
}
static void z_ui_color_parse(const char*s,uint8_t*r,uint8_t*g,uint8_t*b,uint8_t*a){*r=*g=*b=0;*a=255;if(s==NULL||strcmp(s,"transparent")==0){*a=0;return;}if(*s=='#')s++;size_t n=strlen(s);unsigned long v=strtoul(s,NULL,16);if(n==6){*r=(v>>16)&255;*g=(v>>8)&255;*b=v&255;}else if(n==8){*r=(v>>24)&255;*g=(v>>16)&255;*b=(v>>8)&255;*a=v&255;}}
#if defined(__linux__)
static void z_ui_draw_color(SDL_Renderer*r,const char*color){uint8_t red,green,blue,alpha;z_ui_color_parse(color,&red,&green,&blue,&alpha);z_desktop_sdl.SetRenderDrawColor(r,red,green,blue,alpha);}
static void z_ui_fill(SDL_Renderer*r,ZUIRectNative q,const char*color){uint8_t a,b,c,d;z_ui_color_parse(color,&a,&b,&c,&d);if(d==0)return;z_desktop_sdl.SetRenderDrawColor(r,a,b,c,d);ZDesktopSDLFRect f={(float)q.x,(float)q.y,(float)q.width,(float)q.height};z_desktop_sdl.RenderFillRect(r,&f);}
static void z_ui_stroke(SDL_Renderer*r,ZUIRectNative q,const char*color){z_ui_draw_color(r,color);ZDesktopSDLFRect f={(float)q.x,(float)q.y,(float)q.width,(float)q.height};z_desktop_sdl.RenderRect(r,&f);}
static void z_ui_text(SDL_Renderer*r,double x,double y,const char*text,const char*color){if(text==NULL||*text=='\0'||z_desktop_sdl.RenderDebugText==NULL)return;z_ui_draw_color(r,color);z_desktop_sdl.RenderDebugText(r,(float)x,(float)y,text);}
static const char*z_ui_node_text(ZUINodeNative*n){const char*keys[]={"text","label","value","placeholder","title",NULL};for(size_t i=0;keys[i];i++){ZValue v=z_desktop_dict_get(n->props,keys[i]);if(v.tag==ZV_STRING&&*v.as.s)return v.as.s;}return "";}
static const char*z_ui_value_text(ZValue value){if(value.tag==ZV_STRING)return value.as.s;ZValue converted=z_to_string_value(value);return converted.tag==ZV_STRING?converted.as.s:"";}
static void z_ui_render_node(SDL_Renderer*r,ZUINodeNative*n,ZUIContextNative*ctx){
    if(n==NULL||!n->visible) { return; }
    const char*surface=z_ui_theme_string(ctx->theme,"surface","#ffffff"),*text=z_ui_theme_string(ctx->theme,"text","#172033"),*border=z_ui_theme_string(ctx->theme,"border","#cfd6e2"),*primary=z_ui_theme_string(ctx->theme,"primary","#3867e8");
    const char*bg=z_ui_prop_string(n->props,"background","transparent"),*fg=z_ui_prop_string(n->props,"textColor",text);
    ZUIRectNative b=n->bounds;
    if(strcmp(n->kind,"text")==0)z_ui_text(r,b.x+2,b.y+4,z_ui_node_text(n),fg);
    else if(strcmp(n->kind,"button")==0){if(strcmp(bg,"transparent")==0)bg=primary;z_ui_fill(r,b,bg);z_ui_stroke(r,b,border);z_ui_text(r,b.x+10,b.y+(b.height-8)/2,z_ui_node_text(n),z_ui_prop_string(n->props,"textColor","#ffffff"));}
    else if(strcmp(n->kind,"input")==0||strcmp(n->kind,"textarea")==0||strcmp(n->kind,"select")==0){if(strcmp(bg,"transparent")==0)bg=surface;z_ui_fill(r,b,bg);z_ui_stroke(r,b,border);const char*label=z_ui_node_text(n);if(*label=='\0')label=z_ui_prop_string(n->props,"placeholder","");z_ui_text(r,b.x+8,b.y+8,label,fg);if(strcmp(n->kind,"select")==0)z_ui_text(r,b.x+b.width-18,b.y+8,"v",fg);}
    else if(strcmp(n->kind,"checkbox")==0||strcmp(n->kind,"radio")==0){ZUIRectNative box={b.x+4,b.y+(b.height-16)/2,16,16};z_ui_fill(r,box,surface);z_ui_stroke(r,box,border);if(z_ui_prop_bool(n->props,"checked",false))z_ui_fill(r,(ZUIRectNative){box.x+4,box.y+4,8,8},primary);z_ui_text(r,b.x+28,b.y+(b.height-8)/2,z_ui_node_text(n),fg);}
    else if(strcmp(n->kind,"progress")==0){z_ui_fill(r,b,z_ui_prop_string(n->props,"background",z_ui_theme_string(ctx->theme,"surfaceAlt","#eef2f7")));double max=z_ui_prop_number(n->props,"max",100),value=z_ui_prop_number(n->props,"value",0);if(max<=0)max=100;if(value<0)value=0;if(value>max)value=max;ZUIRectNative fill=b;fill.width=b.width*value/max;z_ui_fill(r,fill,z_ui_prop_string(n->props,"fill",primary));}
    else if(strcmp(n->kind,"list")==0||strcmp(n->kind,"tree")==0){z_ui_fill(r,b,bg);z_ui_stroke(r,b,border);ZValue items=z_desktop_dict_get(n->props,"items");double row_height=z_ui_prop_number(n->props,"itemHeight",28);if(items.tag==ZV_ARRAY){for(size_t i=0;i<items.as.array->len;i++){double y=b.y+6+(double)i*row_height;if(y+8>b.y+b.height)break;const char*label=z_ui_value_text(items.as.array->items[i]);if(strcmp(n->kind,"tree")==0){char*line=(char*)z_alloc(strlen(label)+3);sprintf(line,"- %s",label);z_ui_text(r,b.x+8,y,line,fg);}else z_ui_text(r,b.x+8,y,label,fg);}}}
    else if(strcmp(n->kind,"tabs")==0){z_ui_fill(r,b,bg);ZValue tabs=z_desktop_dict_get(n->props,"tabs");int64_t selected=(int64_t)z_ui_prop_number(n->props,"selectedIndex",0);double x=b.x;if(tabs.tag==ZV_ARRAY){for(size_t i=0;i<tabs.as.array->len;i++){const char*label=z_ui_value_text(tabs.as.array->items[i]);double width=(double)strlen(label)*8+24;if(width<72)width=72;ZUIRectNative tab={x,b.y,width,b.height};if((int64_t)i==selected)z_ui_fill(r,tab,z_ui_prop_string(n->props,"selectedBackground",surface));z_ui_stroke(r,tab,border);z_ui_text(r,x+10,b.y+(b.height-8)/2,label,fg);x+=width;}}}
    else if(strcmp(n->kind,"menu")==0){z_ui_fill(r,b,bg);z_ui_stroke(r,b,border);const char*label=z_ui_node_text(n);if(*label=='\0')label="Menu";z_ui_text(r,b.x+10,b.y+(b.height-8)/2,label,fg);}
    else if(strcmp(n->kind,"table")==0){z_ui_fill(r,b,bg);z_ui_stroke(r,b,border);ZValue columns=z_desktop_dict_get(n->props,"columns"),rows=z_desktop_dict_get(n->props,"rows");double row_height=z_ui_prop_number(n->props,"rowHeight",30);size_t count=columns.tag==ZV_ARRAY?columns.as.array->len:1;if(count<1)count=1;double column_width=b.width/(double)count;if(columns.tag==ZV_ARRAY){for(size_t i=0;i<columns.as.array->len;i++){double x=b.x+(double)i*column_width;z_ui_text(r,x+6,b.y+8,z_ui_value_text(columns.as.array->items[i]),fg);if(i>0){z_ui_draw_color(r,border);z_desktop_sdl.RenderLine(r,(float)x,(float)b.y,(float)x,(float)(b.y+b.height));}}}z_ui_draw_color(r,border);z_desktop_sdl.RenderLine(r,(float)b.x,(float)(b.y+row_height),(float)(b.x+b.width),(float)(b.y+row_height));if(rows.tag==ZV_ARRAY){for(size_t i=0;i<rows.as.array->len;i++){double y=b.y+row_height+(double)i*row_height;if(y+8>b.y+b.height)break;z_ui_text(r,b.x+6,y+8,z_ui_value_text(rows.as.array->items[i]),fg);z_ui_draw_color(r,border);z_desktop_sdl.RenderLine(r,(float)b.x,(float)(y+row_height),(float)(b.x+b.width),(float)(y+row_height));}}}
    else if(strcmp(n->kind,"image")==0){const char*path=z_ui_prop_string(n->props,"path","");if(*path&&z_desktop_sdl.IOFromFile&&z_desktop_sdl.LoadBMP_IO&&z_desktop_sdl.CreateTextureFromSurface&&z_desktop_sdl.RenderTexture&&z_desktop_sdl.DestroyTexture){SDL_Surface*s=z_desktop_sdl_load_bmp(path);if(s){SDL_Texture*texture=z_desktop_sdl.CreateTextureFromSurface(r,s);z_desktop_sdl.DestroySurface(s);if(texture){ZDesktopSDLFRect d={(float)b.x,(float)b.y,(float)b.width,(float)b.height};z_desktop_sdl.RenderTexture(r,texture,NULL,&d);z_desktop_sdl.DestroyTexture(texture);}}}}
    else if(strcmp(n->kind,"modal")==0){ZUIRectNative viewport={0,0,b.x*2+b.width,b.y*2+b.height};z_ui_fill(r,viewport,z_ui_prop_string(n->props,"overlay","#00000080"));z_ui_fill(r,b,strcmp(bg,"transparent")==0?surface:bg);z_ui_stroke(r,b,border);}
    else if(strcmp(n->kind,"tooltip")==0){z_ui_fill(r,b,strcmp(bg,"transparent")==0?"#202633":bg);z_ui_text(r,b.x+6,b.y+5,z_ui_node_text(n),z_ui_prop_string(n->props,"textColor","#ffffff"));}
    else if(strcmp(n->kind,"canvas")==0){z_ui_fill(r,b,bg);ZValue commands=z_desktop_dict_get(n->props,"commands");if(commands.tag==ZV_ARRAY){for(size_t i=0;i<commands.as.array->len;i++){ZValue command=commands.as.array->items[i];const char*kind=z_ui_prop_string(command,"kind","");ZValue values=z_desktop_dict_get(command,"values");const char*color=z_ui_prop_string(values,"color",text);double x=b.x+z_ui_prop_number(values,"x",0),y=b.y+z_ui_prop_number(values,"y",0);if(strcmp(kind,"fillRect")==0)z_ui_fill(r,(ZUIRectNative){x,y,z_ui_prop_number(values,"width",10),z_ui_prop_number(values,"height",10)},color);else if(strcmp(kind,"rect")==0)z_ui_stroke(r,(ZUIRectNative){x,y,z_ui_prop_number(values,"width",10),z_ui_prop_number(values,"height",10)},color);else if(strcmp(kind,"line")==0){z_ui_draw_color(r,color);z_desktop_sdl.RenderLine(r,(float)x,(float)y,(float)(b.x+z_ui_prop_number(values,"x2",0)),(float)(b.y+z_ui_prop_number(values,"y2",0)));}else if(strcmp(kind,"text")==0)z_ui_text(r,x,y,z_ui_prop_string(values,"text",""),color);}}}
    else{z_ui_fill(r,b,bg);if(z_ui_prop_bool(n->props,"border",false))z_ui_stroke(r,b,border);}
    if(ctx->focus_id&&strcmp(ctx->focus_id,n->id)==0) {
        z_ui_stroke(r,(ZUIRectNative){b.x-2,b.y-2,b.width+4,b.height+4},z_ui_theme_string(ctx->theme,"focus","#6e95ff"));
    }
    for(size_t i=0;i<n->children.as.array->len;i++) {
        z_ui_render_node(r,z_ui_expect_node(n->children.as.array->items[i]),ctx);
    }
}
#endif
static void z_ui_render_context(ZUIContextNative*ctx){if(ctx==NULL||ctx->closed||ctx->window==NULL||!ctx->window->open)return;z_ui_layout(ctx->root,(ZUIRectNative){0,0,(double)ctx->window->width,(double)ctx->window->height},ctx->theme);
#if defined(__linux__)
    if(!ctx->app->headless){if(ctx->window->renderer==NULL){ctx->window->renderer=z_desktop_sdl.CreateRenderer((SDL_Window*)ctx->window->handle,NULL);if(ctx->window->renderer==NULL)z_fatal("uiRender could not create SDL3 renderer: %s",z_desktop_sdl.GetError());}SDL_Renderer*r=(SDL_Renderer*)ctx->window->renderer;z_ui_draw_color(r,z_ui_theme_string(ctx->theme,"background","#f5f7fb"));z_desktop_sdl.RenderClear(r);z_ui_render_node(r,ctx->root,ctx);z_desktop_sdl.RenderPresent(r);}
#endif
    ctx->dirty=false;}
static void z_ui_render_all(ZDesktopAppNative*app){if(app==NULL)return;for(ZUIContextNative*c=app->ui_contexts;c;c=c->next)if(c->dirty)z_ui_render_context(c);}
static ZUINodeNative*z_ui_hit(ZUINodeNative*n,double x,double y){if(n==NULL||!n->visible)return NULL;for(size_t i=n->children.as.array->len;i>0;i--){ZUINodeNative*h=z_ui_hit(z_ui_expect_node(n->children.as.array->items[i-1]),x,y);if(h)return h;}ZUIRectNative b=n->bounds;if(x>=b.x&&y>=b.y&&x<=b.x+b.width&&y<=b.y+b.height)return n;return NULL;}
static bool z_ui_focusable(ZUINodeNative*n){if(!n||!n->visible||!n->enabled)return false;const char*k=n->kind;return strcmp(k,"button")==0||strcmp(k,"input")==0||strcmp(k,"textarea")==0||strcmp(k,"select")==0||strcmp(k,"checkbox")==0||strcmp(k,"radio")==0||strcmp(k,"tabs")==0||strcmp(k,"menu")==0||z_ui_prop_bool(n->props,"focusable",false);}
static void z_ui_call_prop(ZUINodeNative*n,const char*name,ZValue event){ZValue h=z_desktop_dict_get(n->props,name);if(h.tag==ZV_FUNCTION||h.tag==ZV_BUILTIN||h.tag==ZV_BOUND_METHOD)(void)z_call(h,&event,1);}
static ZUINodeNative*z_ui_find_node(ZUINodeNative*n,const char*id){if(n==NULL)return NULL;if(strcmp(n->id,id)==0)return n;for(size_t i=0;i<n->children.as.array->len;i++){ZUINodeNative*r=z_ui_find_node(z_ui_expect_node(n->children.as.array->items[i]),id);if(r)return r;}return NULL;}
static ZValue z_ui_state_set_value(ZUIStateNative*s,ZValue value){s->value=value;for(ZUIBindingNative*b=s->bindings;b;b=b->next){z_ui_dict_set(b->node->props,b->property,value);if(b->node->context)b->node->context->dirty=true;}for(ZUISubscriberNative*h=s->subscribers;h;h=h->next)(void)z_call(h->handler,&value,1);return value;}
static ZUIStateNative*z_ui_bound_state(ZUINodeNative*n,const char*property){for(ZUIReverseBindingNative*b=n?n->bindings:NULL;b;b=b->next)if(strcmp(b->property,property)==0)return b->state;return NULL;}
static void z_ui_update_bound(ZUINodeNative*n,const char*property,ZValue value){ZUIStateNative*s=z_ui_bound_state(n,property);if(s)z_ui_state_set_value(s,value);}
static void z_ui_collect_focusable_nodes(ZUINodeNative*n,ZUINodeNative**items,size_t*count,size_t cap){if(!n||*count>=cap)return;if(z_ui_focusable(n))items[(*count)++]=n;for(size_t i=0;i<n->children.as.array->len;i++)z_ui_collect_focusable_nodes(z_ui_expect_node(n->children.as.array->items[i]),items,count,cap);}
static void z_ui_utf8_backspace(char*value){size_t n=strlen(value);if(n==0)return;size_t i=n-1;while(i>0&&(((unsigned char)value[i]&0xC0)==0x80))i--;value[i]='\0';}
static void z_ui_dispatch_event(ZUIContextNative*ctx,ZDesktopEventNative*event){
    if(!ctx||ctx->closed||!event) { return; }
    double x=z_ui_prop_number(event->data,"x",0),y=z_ui_prop_number(event->data,"y",0);
    ZUINodeNative*target=NULL;
    if(strcmp(event->type,"mouse_down")==0||strcmp(event->type,"mouse_up")==0||strcmp(event->type,"mouse_move")==0||strcmp(event->type,"mouse_motion")==0||strcmp(event->type,"mouse_button_down")==0||strcmp(event->type,"mouse_button_up")==0)target=z_ui_hit(ctx->root,x,y);else if(ctx->focus_id)target=z_ui_find_node(ctx->root,ctx->focus_id);
    const char*shortcut=z_ui_prop_string(event->data,"shortcut","");if(strcmp(event->type,"key_down")==0&&(strcasecmp(shortcut,"tab")==0||strcasecmp(shortcut,"shift+tab")==0)){ZUINodeNative*nodes[2048];size_t count=0;z_ui_collect_focusable_nodes(ctx->root,nodes,&count,2048);if(count){size_t current=count;for(size_t i=0;i<count;i++)if(ctx->focus_id&&strcmp(nodes[i]->id,ctx->focus_id)==0){current=i;break;}bool reverse=strcasecmp(shortcut,"shift+tab")==0;size_t next;if(current==count)next=reverse?count-1:0;else next=reverse?(current+count-1)%count:(current+1)%count;ctx->focus_id=z_strdup(nodes[next]->id);ctx->dirty=true;}return;}
    if(!target||!target->enabled) { return; }
    ZValue ev=z_desktop_event_value(event);
    bool down=strcmp(event->type,"mouse_down")==0||strcmp(event->type,"mouse_button_down")==0;
    bool up=strcmp(event->type,"mouse_up")==0||strcmp(event->type,"mouse_button_up")==0;
    if(down&&z_ui_focusable(target)){ctx->focus_id=z_strdup(target->id);z_ui_call_prop(target,"onFocus",ev);}if(strcmp(event->type,"mouse_move")==0||strcmp(event->type,"mouse_motion")==0){ctx->hover_id=z_strdup(target->id);z_ui_call_prop(target,"onHover",ev);}
    const char*key=z_ui_prop_string(event->data,"key","");bool activate=up||(strcmp(event->type,"key_down")==0&&(strcasecmp(key,"enter")==0||strcasecmp(key,"space")==0));
    if(activate){if(strcmp(target->kind,"button")==0||strcmp(target->kind,"menu")==0)z_ui_call_prop(target,"onClick",ev);else if(strcmp(target->kind,"checkbox")==0){bool checked=!z_ui_prop_bool(target->props,"checked",false);ZValue value=z_bool(checked);z_ui_dict_set(target->props,"checked",value);z_ui_update_bound(target,"checked",value);z_ui_call_prop(target,"onChange",ev);}else if(strcmp(target->kind,"radio")==0){ZValue value=z_bool(true);z_ui_dict_set(target->props,"checked",value);z_ui_update_bound(target,"checked",value);if(target->parent){for(size_t i=0;i<target->parent->children.as.array->len;i++){ZUINodeNative*sibling=z_ui_expect_node(target->parent->children.as.array->items[i]);if(sibling!=target&&strcmp(sibling->kind,"radio")==0)z_ui_dict_set(sibling->props,"checked",z_bool(false));}}z_ui_call_prop(target,"onChange",ev);}else if(strcmp(target->kind,"select")==0||strcmp(target->kind,"tabs")==0){const char*property=strcmp(target->kind,"tabs")==0?"tabs":"options";ZValue items=z_desktop_dict_get(target->props,property);if(items.tag==ZV_ARRAY&&items.as.array->len>0){int64_t index=(int64_t)z_ui_prop_number(target->props,"selectedIndex",0);index=(index+1)%(int64_t)items.as.array->len;ZValue selected_index=z_int(index),selected_value=z_string(z_ui_value_text(items.as.array->items[index]));z_ui_dict_set(target->props,"selectedIndex",selected_index);z_ui_dict_set(target->props,"value",selected_value);z_ui_update_bound(target,"selectedIndex",selected_index);z_ui_update_bound(target,"value",selected_value);z_ui_call_prop(target,"onChange",ev);}}}
    if(strcmp(event->type,"key_down")==0&&(strcmp(target->kind,"input")==0||strcmp(target->kind,"textarea")==0)&&strcasecmp(key,"backspace")==0){const char*current=z_ui_prop_string(target->props,"value","");char*next=z_strdup(current);z_ui_utf8_backspace(next);ZValue value=z_string(next);z_ui_dict_set(target->props,"value",value);z_ui_update_bound(target,"value",value);z_ui_call_prop(target,"onInput",ev);z_ui_call_prop(target,"onChange",ev);}
    if(strcmp(event->type,"text_input")==0&&(strcmp(target->kind,"input")==0||strcmp(target->kind,"textarea")==0)){const char*current=z_ui_prop_string(target->props,"value",""),*added=z_ui_prop_string(event->data,"text","");size_t length=strlen(current)+strlen(added)+1;char*next=(char*)z_alloc(length);snprintf(next,length,"%s%s",current,added);ZValue value=z_string(next);z_ui_dict_set(target->props,"value",value);z_ui_update_bound(target,"value",value);z_ui_call_prop(target,"onInput",ev);z_ui_call_prop(target,"onChange",ev);}ctx->dirty=true;
}
static void z_ui_dispatch_from_desktop(ZDesktopAppNative*app,ZDesktopEventNative*event){if(!app||!event)return;for(ZUIContextNative*c=app->ui_contexts;c;c=c->next)if(c->window&&c->window->id==event->window_id)z_ui_dispatch_event(c,event);}
static ZValue z_ui_snapshot_node(ZUINodeNative*n){ZValue pairs[6]={z_pair(z_string("id"),z_string(n->id)),z_pair(z_string("kind"),z_string(n->kind)),z_pair(z_string("x"),z_float(n->bounds.x)),z_pair(z_string("y"),z_float(n->bounds.y)),z_pair(z_string("width"),z_float(n->bounds.width)),z_pair(z_string("height"),z_float(n->bounds.height))};return z_dict_from(pairs,6);}
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
    if(strcmp(name,"uiUnmount")==0){z_expect_args(name,argc,1);ZUIContextNative*c=z_ui_expect_context(args[0]);bool changed=!c->closed;c->closed=true;return z_bool(changed);}
    if(strcmp(name,"uiRender")==0){z_expect_args(name,argc,1);z_ui_render_context(z_ui_expect_context(args[0]));return z_bool(true);}
    if(strcmp(name,"uiSnapshot")==0){z_expect_args(name,argc,1);return z_ui_snapshot(z_ui_expect_context(args[0]));}
    if(strcmp(name,"uiSetTheme")==0){z_expect_args(name,argc,2);ZUIContextNative*c=z_ui_expect_context(args[0]);c->theme=z_ui_expect_theme(args[1]);c->dirty=true;z_ui_render_context(c);return args[0];}
    if(strcmp(name,"uiDispatch")==0){z_expect_args(name,argc,2);ZUIContextNative*c=z_ui_expect_context(args[0]);if(args[1].tag!=ZV_DICT)z_fatal("uiDispatch event must be dictionary");ZDesktopEventNative e={0};e.type=z_strdup(z_ui_prop_string(args[1],"type","custom"));e.window_id=c->window->id;e.timestamp=z_desktop_now_ns();e.data=args[1];z_ui_dispatch_event(c,&e);z_ui_render_context(c);return z_bool(true);}
    if(strcmp(name,"uiSet")==0){z_expect_args(name,argc,3);ZUINodeNative*n=z_ui_expect_node(args[0]);const char*property=z_as_cstring(args[1]);z_ui_dict_set(n->props,property,args[2]);if(strcmp(property,"visible")==0&&args[2].tag==ZV_BOOL)n->visible=args[2].as.b;if(strcmp(property,"disabled")==0&&args[2].tag==ZV_BOOL)n->enabled=!args[2].as.b;if(n->context)n->context->dirty=true;return args[0];}
    if(strcmp(name,"uiGet")==0){z_expect_args(name,argc,2);ZUINodeNative*n=z_ui_expect_node(args[0]);const char*property=z_as_cstring(args[1]);if(strcmp(property,"id")==0)return z_string(n->id);if(strcmp(property,"kind")==0)return z_string(n->kind);return z_desktop_dict_get(n->props,property);}
    if(strcmp(name,"uiAdd")==0){z_expect_args(name,argc,2);ZUINodeNative*p=z_ui_expect_node(args[0]),*child=z_ui_expect_node(args[1]);size_t len=p->children.as.array->len;ZValue*items=(ZValue*)z_alloc(sizeof(ZValue)*(len+1));if(len)memcpy(items,p->children.as.array->items,sizeof(ZValue)*len);items[len]=args[1];p->children=z_array_from(items,len+1);child->parent=p;if(p->context)z_ui_attach_tree(p->context,child);if(p->context)p->context->dirty=true;return args[0];}
    if(strcmp(name,"uiRemove")==0){z_expect_args(name,argc,2);ZUINodeNative*p=z_ui_expect_node(args[0]);const char*id=z_as_cstring(args[1]);size_t len=p->children.as.array->len;ZValue*items=(ZValue*)z_alloc(sizeof(ZValue)*len);size_t count=0;bool removed=false;for(size_t i=0;i<len;i++){ZUINodeNative*c=z_ui_expect_node(p->children.as.array->items[i]);if(strcmp(c->id,id)==0){removed=true;c->parent=NULL;}else items[count++]=p->children.as.array->items[i];}p->children=z_array_from(items,count);if(p->context)p->context->dirty=true;return z_bool(removed);}
    if(strcmp(name,"uiFind")==0){z_expect_args(name,argc,2);ZUINodeNative*n=z_ui_find_node(z_ui_expect_context(args[0])->root,z_as_cstring(args[1]));return n?z_ui_value(ZV_UI_NODE,ZK_UI_NODE,n):z_null();}
    if(strcmp(name,"uiFocus")==0){z_expect_args(name,argc,2);ZUIContextNative*c=z_ui_expect_context(args[0]);ZUINodeNative*n=z_ui_expect_node(args[1]);if(!z_ui_focusable(n))return z_bool(false);c->focus_id=z_strdup(n->id);c->dirty=true;return z_bool(true);}
    if(strcmp(name,"uiFocusNext")==0){z_expect_args(name,argc,2);ZUIContextNative*c=z_ui_expect_context(args[0]);bool reverse=args[1].tag==ZV_BOOL&&args[1].as.b;ZUINodeNative*nodes[2048];size_t count=0;z_ui_collect_focusable_nodes(c->root,nodes,&count,2048);if(count==0)return z_null();size_t current=count;for(size_t i=0;i<count;i++)if(c->focus_id&&strcmp(nodes[i]->id,c->focus_id)==0){current=i;break;}size_t next=current==count?(reverse?count-1:0):(reverse?(current+count-1)%count:(current+1)%count);c->focus_id=z_strdup(nodes[next]->id);c->dirty=true;return z_ui_value(ZV_UI_NODE,ZK_UI_NODE,nodes[next]);}
    if(strcmp(name,"uiAccessibility")==0){z_expect_args(name,argc,1);ZValue items[2048];size_t count=0;z_ui_collect_accessibility(z_ui_expect_context(args[0])->root,items,&count,2048);return z_array_from(items,count);}
    if(strcmp(name,"uiCanvasCommand")==0){z_expect_args(name,argc,3);ZUINodeNative*n=z_ui_expect_node(args[0]);if(strcmp(n->kind,"canvas")!=0)z_fatal("uiCanvasCommand expects canvas");ZValue commands=z_desktop_dict_get(n->props,"commands");size_t len=commands.tag==ZV_ARRAY?commands.as.array->len:0;ZValue*items=(ZValue*)z_alloc(sizeof(ZValue)*(len+1));if(len)memcpy(items,commands.as.array->items,sizeof(ZValue)*len);ZValue pairs[2]={z_pair(z_string("kind"),args[1]),z_pair(z_string("values"),args[2])};items[len]=z_dict_from(pairs,2);z_ui_dict_set(n->props,"commands",z_array_from(items,len+1));if(n->context)n->context->dirty=true;return args[0];}
    if(handled) { *handled=false; }
    return z_null();
}

#endif
