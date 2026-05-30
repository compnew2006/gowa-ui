"use client";

import { useState, useEffect, useCallback } from "react";
import { 
  Plus, 
  Calendar, 
  Clock, 
  Facebook, 
  Instagram, 
  MessageCircle, 
  Trash2, 
  Send,
  Sparkles,
  Loader2,
  CheckCircle2,
  AlertCircle
} from "lucide-react";
import { api, ApiError, type Post } from "@/lib/api";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { RequirePermission } from "@/components/auth/permission-guard";
import { toast } from "sonner";
import { format } from "date-fns";

export default function PostsPage() {
  const [posts, setPosts] = useState<Post[]>([]);
  const [pages, setPages] = useState<{ id: string; name: string; platform: string; is_active: boolean; avatar_url?: string }[]>([]);
  const [loading, setLoading] = useState(true);
  const [isCreating, setIsCreating] = useState(false);
  
  // Form state
  const [pageId, setPageId] = useState("");
  const [platform, setPlatform] = useState("facebook");
  const [message, setMessage] = useState("");
  const [scheduledAt, setScheduledAt] = useState("");
  const [isGenerating, setIsGenerating] = useState(false);
  const [tone, setTone] = useState("Professional");
  const [deleteId, setDeleteId] = useState<string | null>(null);

  const selectedPage = pages.find(p => p.id === pageId);
  const pageName = selectedPage ? selectedPage.name : "العلامة التجارية";
  const avatarUrl = selectedPage?.avatar_url || "https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?w=150&auto=format&fit=crop&q=60";

  const loadData = useCallback(async () => {
    try {
      const [postsData, pagesData] = await Promise.all([
        api.getPosts(),
        api.getPages()
      ]);
      setPosts(postsData);
      setPages(pagesData.filter(p => p.is_active));
      if (pagesData.length > 0) setPageId(pagesData[0].id);
    } catch {
      toast.error("Failed to load posts");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleCreatePost = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!pageId || !message || !scheduledAt) {
      toast.error("Please fill all fields");
      return;
    }

    try {
      await api.createPost({
        page_id: pageId,
        platform,
        message,
        scheduled_at: new Date(scheduledAt).toISOString(),
      });
      toast.success("Post scheduled successfully");
      setIsCreating(false);
      setMessage("");
      loadData();
    } catch {
      toast.error("Failed to schedule post");
    }
  };

  const handleDeletePost = async (id: string) => {
    try {
      await api.deletePost(id);
      toast.success("Post deleted");
      setDeleteId(null);
      loadData();
    } catch {
      toast.error("Failed to delete post");
    }
  };

  const generateAIContent = async () => {
    setIsGenerating(true);
    try {
      const result = await api.generatePostContent({
        page_id: pageId,
        platform,
        language: "ar",
        prompt: message || undefined,
        tone: tone,
      });
      setMessage(result.message);
      toast.success("AI content generated!");
    } catch (error) {
      if (error instanceof ApiError && [404, 501].includes(error.status)) {
        toast.error("AI generation endpoint is not available yet");
      } else {
        toast.error("AI generation failed");
      }
    } finally {
      setIsGenerating(false);
    }
  };

  return (
    <RequirePermission permission="can_manage_campaigns">
    <div className="p-8 space-y-8 animate-in fade-in duration-500">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-indigo-600">
            Social Media Poster
          </h1>
          <p className="text-muted-foreground mt-1">Schedule and manage your feed posts across platforms.</p>
        </div>
        <button 
          onClick={() => setIsCreating(true)}
          className="flex items-center gap-2 bg-indigo-600 hover:bg-indigo-700 text-white px-6 py-2.5 rounded-xl transition-all shadow-lg shadow-indigo-500/20 active:scale-95"
        >
          <Plus className="w-5 h-5" />
          Schedule Post
        </button>
      </div>

      {isCreating && (
        <div className="bg-card/40 backdrop-blur-xl border border-white/10 rounded-3xl p-6 shadow-2xl animate-in slide-in-from-top duration-300">
            <form onSubmit={handleCreatePost} className="grid grid-cols-1 lg:grid-cols-2 gap-8">
              {/* Left Column: Controls */}
              <div className="space-y-5">
                <div className="border-b border-white/5 pb-3">
                  <h3 className="text-lg font-semibold bg-clip-text text-transparent bg-gradient-to-r from-indigo-400 to-purple-400">إعدادات المنشور</h3>
                  <p className="text-xs text-muted-foreground mt-0.5">حدد الصفحة والمنصة ووقت النشر ونبرة الصوت المطلوبة.</p>
                </div>

                <div>
                  <label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-2 block">الصفحة المستهدفة (Target Page)</label>
                  <select 
                    value={pageId}
                    onChange={(e) => setPageId(e.target.value)}
                    className="w-full bg-background border border-white/10 rounded-xl px-4 py-2.5 focus:ring-2 focus:ring-indigo-500 transition-all outline-none text-sm cursor-pointer"
                  >
                    {pages.map(page => (
                      <option key={page.id} value={page.id}>{page.name} ({page.platform})</option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-2 block">المنصة المستهدفة (Platform)</label>
                  <div className="flex gap-3">
                    {['facebook', 'instagram', 'whatsapp'].map(p => (
                      <button
                        key={p}
                        type="button"
                        onClick={() => setPlatform(p)}
                        className={`flex-1 flex items-center justify-center gap-2 py-2.5 rounded-xl border transition-all ${
                          platform === p 
                            ? "bg-indigo-500/10 border-indigo-500 text-indigo-400 shadow-inner" 
                            : "border-white/5 hover:border-white/20 bg-background/50"
                        }`}
                      >
                        {p === 'facebook' && <Facebook className="w-4 h-4" />}
                        {p === 'instagram' && <Instagram className="w-4 h-4" />}
                        {p === 'whatsapp' && <MessageCircle className="w-4 h-4" />}
                        <span className="capitalize text-xs font-medium">{p}</span>
                      </button>
                    ))}
                  </div>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div>
                    <label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-2 block">نبرة الصوت للذكاء الاصطناعي</label>
                    <select
                      value={tone}
                      onChange={(e) => setTone(e.target.value)}
                      className="w-full bg-background border border-white/10 rounded-xl px-4 py-2.5 focus:ring-2 focus:ring-indigo-500 transition-all outline-none text-sm cursor-pointer"
                    >
                      <option value="Professional">مهني / احترافي (Professional)</option>
                      <option value="Friendly">ودود / لطيف (Friendly)</option>
                      <option value="Bold">جريء / قوي (Bold)</option>
                      <option value="Humorous">مرح / فكاهي (Humorous)</option>
                      <option value="Empathetic">متعاطف / داعم (Empathetic)</option>
                    </select>
                  </div>
                  <div>
                    <label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-2 block">وقت النشر المجدول</label>
                    <input 
                      type="datetime-local" 
                      value={scheduledAt}
                      onChange={(e) => setScheduledAt(e.target.value)}
                      className="w-full bg-background border border-white/10 rounded-xl px-4 py-2 focus:ring-2 focus:ring-indigo-500 transition-all outline-none text-sm font-mono"
                    />
                  </div>
                </div>

                <div className="flex gap-3 pt-4 border-t border-white/5">
                  <button 
                    type="submit"
                    className="flex-1 bg-indigo-600 hover:bg-indigo-700 text-white py-3 rounded-xl transition-all font-semibold flex items-center justify-center gap-2 active:scale-95 shadow-lg shadow-indigo-500/20"
                  >
                    <Send className="w-4 h-4" />
                    جدولة ونشر المنشور
                  </button>
                  <button 
                    type="button"
                    onClick={() => setIsCreating(false)}
                    className="px-6 py-3 border border-white/10 rounded-xl hover:bg-white/5 transition-all text-sm font-medium"
                  >
                    إلغاء
                  </button>
                </div>
              </div>

              {/* Right Column: Text editor & Live Preview */}
              <div className="space-y-5">
                <div className="relative">
                  <label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-2 flex justify-between items-center">
                    <span>محتوى المنشور (Caption Copy)</span>
                    <button 
                      type="button"
                      onClick={generateAIContent}
                      disabled={isGenerating}
                      className="text-xs text-indigo-400 hover:text-indigo-300 flex items-center gap-1 transition-colors disabled:opacity-50 font-medium bg-indigo-500/5 px-2.5 py-1 rounded-lg border border-indigo-500/10 active:scale-95"
                    >
                      {isGenerating ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Sparkles className="w-3.5 h-3.5" />}
                      توليد إبداعي بالذكاء الاصطناعي (AI)
                    </button>
                  </label>
                  <textarea 
                    rows={4}
                    value={message}
                    onChange={(e) => setMessage(e.target.value)}
                    placeholder="اكتب فكرة منشورك هنا، أو انقر على 'توليد بالذكاء الاصطناعي' ليقوم المساعد بكتابة منشور مخصص متوافق تماماً مع هوية علامتك التجارية..."
                    className="w-full bg-background border border-white/10 rounded-xl px-4 py-3 focus:ring-2 focus:ring-indigo-500 transition-all outline-none resize-none text-sm leading-relaxed"
                  />
                </div>

                {/* Live Platform Preview Cards */}
                <div className="space-y-3">
                  <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground block">المعاينة الحية التفاعلية (Live Mockup)</span>
                  
                  {platform === "facebook" && (
                    <div className="bg-[#18191a] text-[#e4e6eb] border border-white/10 rounded-2xl p-4 shadow-xl font-sans w-full max-w-md mx-auto animate-in fade-in duration-300">
                      <div className="flex items-center gap-3">
                        <img src={avatarUrl} alt="Avatar" className="w-10 h-10 rounded-full border border-white/5 object-cover" />
                        <div>
                          <h4 className="text-sm font-bold text-white leading-none">{pageName}</h4>
                          <span className="text-[10px] text-[#b0b3b8] flex items-center gap-1 mt-1 font-mono">
                            Just Now · 🌎
                          </span>
                        </div>
                      </div>
                      <p className="mt-3 text-sm whitespace-pre-wrap leading-relaxed text-[#e4e6eb]/90">{message || "سيظهر محتوى منشورك المولد هنا..."}</p>
                      <div className="mt-4 aspect-video bg-gradient-to-br from-indigo-950/60 to-purple-950/60 border border-white/5 rounded-xl flex flex-col items-center justify-center p-4 text-center gap-2 relative overflow-hidden group">
                        <Sparkles className="w-6 h-6 text-indigo-400 animate-pulse relative z-10" />
                        <span className="text-[10px] text-indigo-300 uppercase tracking-widest font-mono font-bold relative z-10">Creative Media Concept</span>
                        <p className="text-xs text-white/70 italic px-4 line-clamp-2 relative z-10">"تم توليد محتوى المنشور بذكاء لتطبيقه مع هوية علامتك التجارية المحددة مسبقاً"</p>
                        <div className="absolute inset-0 bg-radial-gradient from-transparent to-[#000]/30"></div>
                      </div>
                      <div className="mt-4 pt-3 border-t border-white/10 flex justify-around text-xs text-[#b0b3b8] font-medium">
                        <button type="button" className="hover:text-white flex items-center gap-1.5 py-1 px-3 hover:bg-white/5 rounded-lg transition-all"><span className="scale-110">👍</span> Like</button>
                        <button type="button" className="hover:text-white flex items-center gap-1.5 py-1 px-3 hover:bg-white/5 rounded-lg transition-all"><span className="scale-110">💬</span> Comment</button>
                        <button type="button" className="hover:text-white flex items-center gap-1.5 py-1 px-3 hover:bg-white/5 rounded-lg transition-all"><span className="scale-110">➡️</span> Share</button>
                      </div>
                    </div>
                  )}

                  {platform === "instagram" && (
                    <div className="bg-[#000000] text-white border border-white/10 rounded-2xl p-4 shadow-xl font-sans w-full max-w-md mx-auto animate-in fade-in duration-300">
                      <div className="flex items-center justify-between pb-3 border-b border-white/5">
                        <div className="flex items-center gap-3">
                          <img src={avatarUrl} alt="Avatar" className="w-8 h-8 rounded-full border border-white/10 object-cover" />
                          <span className="text-xs font-bold">{pageName.toLowerCase().replace(/\s+/g, '_')}</span>
                        </div>
                        <span className="text-xs text-white/50 tracking-widest font-bold cursor-pointer">•••</span>
                      </div>
                      <div className="mt-3 aspect-square bg-gradient-to-tr from-pink-950/40 via-purple-950/40 to-indigo-950/40 border border-white/5 rounded-xl flex flex-col items-center justify-center p-6 text-center gap-2 relative overflow-hidden">
                        <Sparkles className="w-8 h-8 text-pink-400 animate-pulse relative z-10" />
                        <span className="text-[10px] text-pink-300 uppercase tracking-widest font-mono font-bold relative z-10">AI Creative Prompt</span>
                        <p className="text-xs text-white/80 italic px-2 line-clamp-3 relative z-10">"توليد صورة فنية تعبيرية تتناسب مع نبرة وهوية الماركة"</p>
                        <div className="absolute inset-0 bg-radial-gradient from-transparent to-[#000]/60 opacity-60"></div>
                      </div>
                      <div className="flex justify-between items-center mt-3 text-lg">
                        <div className="flex gap-4">
                          <button type="button" className="hover:scale-110 transition-transform">❤️</button>
                          <button type="button" className="hover:scale-110 transition-transform">💬</button>
                          <button type="button" className="hover:scale-110 transition-transform">✈️</button>
                        </div>
                        <button type="button" className="hover:scale-110 transition-transform">📥</button>
                      </div>
                      <p className="mt-3 text-xs leading-relaxed text-white/95">
                        <span className="font-bold mr-2">{pageName.toLowerCase().replace(/\s+/g, '_')}</span>
                        {message || "سيظهر محتوى منشورك المولد هنا..."}
                      </p>
                    </div>
                  )}

                  {platform === "whatsapp" && (
                    <div className="bg-[#0b141a] text-[#e9edef] border border-white/10 rounded-2xl p-4 shadow-xl font-sans w-full max-w-md mx-auto animate-in fade-in duration-300" dir="ltr">
                      <div className="flex items-center gap-3 pb-2 border-b border-white/5">
                        <img src={avatarUrl} alt="Avatar" className="w-9 h-9 rounded-full object-cover" />
                        <div>
                          <h4 className="text-sm font-bold text-white">{pageName}</h4>
                          <span className="text-[10px] text-[#8696a0] font-mono">online</span>
                        </div>
                      </div>
                      <div className="mt-4 bg-[#202c33] rounded-2xl rounded-tl-none p-3 max-w-[85%] relative self-start border border-white/5">
                        <p className="text-sm whitespace-pre-wrap leading-relaxed text-[#e9edef]">{message || "سيظهر محتوى منشورك المولد هنا..."}</p>
                        <div className="flex justify-end items-center gap-1 mt-1.5 text-[9px] text-[#8696a0] font-mono">
                          <span>12:00 PM</span>
                          <span className="text-[#53bdeb] text-[10px]">✓✓</span>
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </form>
          </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Scheduled List */}
        <div className="lg:col-span-2 space-y-4">
          <div className="flex items-center gap-2 text-lg font-semibold px-2">
            <Calendar className="w-5 h-5 text-indigo-400" />
            Scheduled Queue
          </div>
          
          {loading ? (
            <div className="flex flex-col items-center justify-center py-20 gap-4">
              <Loader2 className="w-8 h-8 animate-spin text-indigo-500" />
              <p className="text-muted-foreground animate-pulse">Loading your queue...</p>
            </div>
          ) : posts.length === 0 ? (
            <div className="bg-card/30 border border-dashed border-white/10 rounded-2xl py-20 flex flex-col items-center justify-center gap-4">
              <div className="p-4 bg-white/5 rounded-full">
                <Clock className="w-10 h-10 text-muted-foreground" />
              </div>
              <p className="text-muted-foreground">No posts scheduled yet.</p>
            </div>
          ) : (
            <div className="grid gap-4">
              {posts.map(post => (
                <div key={post.id} className="group bg-card/40 hover:bg-card/60 border border-white/5 hover:border-indigo-500/30 rounded-xl p-5 transition-all shadow-sm">
                  <div className="flex justify-between items-start gap-4">
                    <div className="flex-1">
                      <div className="flex items-center gap-3 mb-3">
                        <div className={`p-2 rounded-lg ${
                          post.platform === 'facebook' ? 'bg-blue-500/10 text-blue-400' :
                          post.platform === 'instagram' ? 'bg-pink-500/10 text-pink-400' :
                          'bg-green-500/10 text-green-400'
                        }`}>
                          {post.platform === 'facebook' && <Facebook className="w-4 h-4" />}
                          {post.platform === 'instagram' && <Instagram className="w-4 h-4" />}
                          {post.platform === 'whatsapp' && <MessageCircle className="w-4 h-4" />}
                        </div>
                        <div>
                          <p className="text-xs text-muted-foreground uppercase tracking-wider font-semibold">
                            {pages.find(p => p.id === post.page_id)?.name || 'Unknown Page'}
                          </p>
                          <p className="text-sm font-medium flex items-center gap-1.5">
                            <Clock className="w-3.5 h-3.5" />
                            {format(new Date(post.scheduled_at), "PPP p")}
                          </p>
                        </div>
                      </div>
                      <p className="text-sm leading-relaxed line-clamp-3 text-foreground/80">
                        {post.message}
                      </p>
                    </div>
                    
                    <div className="flex flex-col items-end gap-3">
                      <span className={`px-2.5 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-widest ${
                        post.status === 'posted' ? 'bg-green-500/10 text-green-400' :
                        post.status === 'pending' ? 'bg-amber-500/10 text-amber-400' :
                        'bg-red-500/10 text-red-400'
                      }`}>
                        {post.status}
                      </span>
                      <button 
                        onClick={() => setDeleteId(post.id)}
                        className="p-2 text-muted-foreground hover:text-red-400 hover:bg-red-400/10 rounded-lg transition-all opacity-0 group-hover:opacity-100"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </div>
                  
                  {post.error && (
                    <div className="mt-4 p-2.5 bg-red-500/5 border border-red-500/10 rounded-lg flex items-center gap-2 text-xs text-red-400">
                      <AlertCircle className="w-3.5 h-3.5" />
                      {post.error}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Analytics/Summary Sidebar */}
        <div className="space-y-6">
          <div className="bg-card/30 border border-white/10 rounded-2xl p-6">
            <h3 className="font-semibold mb-4 flex items-center gap-2">
              <CheckCircle2 className="w-5 h-5 text-green-400" />
              Posting Stats
            </h3>
            <div className="space-y-4">
              <div className="flex justify-between items-center p-3 bg-white/5 rounded-xl">
                <span className="text-sm text-muted-foreground">Total Scheduled</span>
                <span className="font-bold">{posts.filter(p => p.status === 'pending').length}</span>
              </div>
              <div className="flex justify-between items-center p-3 bg-white/5 rounded-xl">
                <span className="text-sm text-muted-foreground">Successfully Posted</span>
                <span className="font-bold">{posts.filter(p => p.status === 'posted').length}</span>
              </div>
              <div className="flex justify-between items-center p-3 bg-white/5 rounded-xl">
                <span className="text-sm text-muted-foreground">Failed Attempts</span>
                <span className="font-bold text-red-400">{posts.filter(p => p.status === 'failed').length}</span>
              </div>
            </div>
          </div>

          <div className="bg-gradient-to-br from-indigo-500/10 to-purple-500/10 border border-white/10 rounded-2xl p-6 relative overflow-hidden group">
            <div className="relative z-10">
              <h3 className="font-semibold mb-2 flex items-center gap-2">
                <Sparkles className="w-5 h-5 text-amber-400" />
                AI Strategy
              </h3>
              <p className="text-xs text-muted-foreground leading-relaxed">
                Our AI analyzes your audience engagement to recommend the best posting times.
              </p>
              <button className="mt-4 w-full py-2 bg-white/10 hover:bg-white/20 rounded-lg text-xs font-medium transition-all">
                View Recommendations
              </button>
            </div>
            <div className="absolute -right-4 -bottom-4 opacity-5 group-hover:scale-110 transition-transform duration-700">
              <Sparkles className="w-24 h-24" />
            </div>
          </div>
        </div>
      </div>
      <ConfirmDialog
        open={Boolean(deleteId)}
        title="Delete scheduled post"
        description="This removes the scheduled post from the queue. This action cannot be undone."
        destructive
        confirmLabel="Delete"
        onCancel={() => setDeleteId(null)}
        onConfirm={() => deleteId && handleDeletePost(deleteId)}
      />
    </div>
    </RequirePermission>
  );
}
