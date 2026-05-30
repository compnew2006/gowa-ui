#!/usr/bin/env python3
"""
اختبار شامل لنظام SQLite للتعليقات
"""
import sys
import os
sys.path.insert(0, '/opt/hermes-webhook')

from facebook_db import FacebookPageDB
from datetime import datetime

def test_database():
    """اختبار قاعدة البيانات"""
    print("🧪 اختبار قاعدة البيانات SQLite")
    print("=" * 40)
    
    page_id = "895247390337022"
    
    # إنشاء قاعدة بيانات
    print(f"📁 إنشاء قاعدة بيانات للصفحة {page_id}...")
    db = FacebookPageDB(page_id)
    
    # إضافة تعليق تجريبي
    print("➕ إضافة تعليق تجريبي...")
    test_comment = {
        'comment_id': 'test_' + str(int(datetime.now().timestamp())),
        'post_id': 'post_test_123',
        'message': 'يا هلا بالطيب كيف حالك هل عندك ادارة صفحات يوتيوب؟؟',
        'from': 'Nomani Mostafa',
        'sender_id': '26220352977614710',
        'verb': 'add',
        'created_time': int(datetime.now().timestamp()),
        'raw': {}
    }
    
    comment_id = db.add_comment(test_comment)
    print(f"✅ تم إضافة التعليق (DB ID: {comment_id})")
    
    # اختبار حفظ الرد
    print("➕ إضافة رد تلقائي...")
    db.mark_as_replied(
        test_comment['comment_id'],
        'reply_test_456',
        'أهلاً وسهلاً! نعم نقدم خدمات إدارة صفحات يوتيوب. تواصل معنا للتفاصيل!',
        ai_model='test_model',
        reply_type='auto'
    )
    print("✅ تم حفظ الرد")
    
    # عرض الإحصائيات
    print("\n📊 الإحصائيات:")
    stats = db.get_stats()
    for key, value in stats.items():
        print(f"  • {key}: {value}")
    
    # البحث
    print("\n🔍 اختبار البحث عن 'يوتيوب'...")
    results = db.search_comments('يوتيوب')
    print(f"✅ تم العثور على {len(results)} نتيجة")
    
    # التعليقات الأخيرة
    print("\n📝 التعليقات الأخيرة:")
    recent = db.get_recent_comments(hours=1, limit=3)
    for i, comment in enumerate(recent, 1):
        status = "✅" if comment['replied'] else "⏳"
        print(f"  {i}. {status} {comment['from_name']}: {comment['message'][:50]}")
    
    print("\n" + "=" * 40)
    print("✅ نجح الاختبار!")
    print(f"📍 موقع قاعدة البيانات: /opt/hermes-webhook/databases/page_{page_id}.db")

if __name__ == "__main__":
    try:
        test_database()
    except Exception as e:
        print(f"❌ فشل الاختبار: {e}")
        import traceback
        traceback.print_exc()
