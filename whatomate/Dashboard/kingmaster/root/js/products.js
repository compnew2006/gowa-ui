// تحميل المنتجات عند فتح الصفحة
document.addEventListener('DOMContentLoaded', function() {
    loadProducts();
    
    // الاستماع للتغييرات في الفلاتر
    document.getElementById('categoryFilter').addEventListener('change', loadProducts);
    document.getElementById('typeFilter').addEventListener('change', loadProducts);
    document.getElementById('minPrice').addEventListener('input', debounce(loadProducts, 500));
    document.getElementById('maxPrice').addEventListener('input', debounce(loadProducts, 500));
    
    // البحث عند الضغط على Enter
    document.getElementById('searchInput').addEventListener('keypress', function(e) {
        if (e.key === 'Enter') {
            loadProducts();
        }
    });
});

// دالة التأخير للبحث
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

// جلب المنتجات من API
async function loadProducts() {
    const search = document.getElementById('searchInput').value;
    const category = document.getElementById('categoryFilter').value;
    const isDigital = document.getElementById('typeFilter').value;
    const minPrice = document.getElementById('minPrice').value || 0;
    const maxPrice = document.getElementById('maxPrice').value || 999999;
    
    const params = new URLSearchParams({
        search: search,
        category: category,
        is_digital: isDigital,
        min_price: minPrice,
        max_price: maxPrice
    });
    
    try {
        const response = await fetch(`api/products.php?${params}`);
        const data = await response.json();
        
        if (data.success) {
            renderProducts(data.products);
        } else {
            showError(data.message);
        }
    } catch (error) {
        console.error('خطأ في جلب المنتجات:', error);
        showError('حدث خطأ في جلب المنتجات');
    }
}

// عرض المنتجات
function renderProducts(products) {
    const grid = document.getElementById('productsGrid');
    
    if (products.length === 0) {
        grid.innerHTML = `
            <div class="empty-state" style="grid-column: 1/-1;">
                <i class="fas fa-box-open"></i>
                <h3>لا توجد منتجات</h3>
                <p>لم يتم العثور على منتجات مطابقة للبحث</p>
            </div>
        `;
        return;
    }
    
    grid.innerHTML = products.map(product => createProductCard(product)).join('');
}

// إنشاء بطاقة منتج
function createProductCard(product) {
    const isDigital = product.is_digital;
    const actualStock = product.stock_quantity !== undefined && product.stock_quantity !== null ? product.stock_quantity : product.stock;
    const stockStatus = getStockStatus(actualStock);
    const productIcon = getProductIcon(product.category, isDigital);
    
    return `
        <div class="product-card" onclick="openProductDetails(${product.id})">
            <div class="product-image">
                ${product.image && product.image !== '' ? 
                    `<img src="uploads/products/${product.image}" alt="${product.name}" onerror="this.style.display='none'; this.nextElementSibling.style.display='flex';">` :
                    ''
                }
                <i class="${productIcon}" style="${product.image && product.image !== '' ? 'display: none;' : ''}"></i>
                <div class="product-badge ${isDigital ? 'badge-digital' : 'badge-physical'}">
                    <i class="fas ${isDigital ? 'fa-download' : 'fa-box'}"></i>
                    ${isDigital ? 'رقمي' : 'حقيقي'}
                </div>
                ${stockStatus.html}
            </div>
            
            <div class="product-info">
                <div class="product-name">${product.name}</div>
                <div class="product-description">${product.description || 'لا يوجد وصف'}</div>
                
                <div class="product-meta">
                    ${product.category ? `
                        <span class="meta-badge">
                            <i class="fas fa-tag"></i>
                            ${getCategoryName(product.category)}
                        </span>
                    ` : ''}
                    ${product.colors_array && product.colors_array.length > 0 ? `
                        <span class="meta-badge">
                            <i class="fas fa-palette"></i>
                            ${product.colors_array.length} لون
                        </span>
                    ` : ''}
                    ${product.sizes_array && product.sizes_array.length > 0 ? `
                        <span class="meta-badge">
                            <i class="fas fa-ruler"></i>
                            ${product.sizes_array.length} مقاس
                        </span>
                    ` : ''}
                </div>
                
                <div class="product-footer">
                    <div>
                        <div class="product-price">
                            <i class="fas fa-coins"></i>
                            ${formatPrice(product.price)}
                        </div>
                        ${product.commission > 0 ? `
                            <div style="font-size: 12px; color: #27ae60; font-weight: 600; margin-top: 5px; display: flex; align-items: center; gap: 5px; font-family: 'Cairo', sans-serif;">
                                <i class="fas fa-hand-holding-usd"></i>
                                عمولة: ${formatPrice(product.commission)}
                            </div>
                        ` : ''}
                    </div>
                    <button class="buy-btn" onclick="event.stopPropagation(); openProductDetails(${product.id})">
                        <i class="fas fa-shopping-cart"></i>
                        شراء
                    </button>
                </div>
            </div>
        </div>
    `;
}

// الحصول على حالة المخزون
function getStockStatus(stock) {
    if (stock === 0) {
        return {
            html: '<div class="stock-badge stock-out"><i class="fas fa-times"></i> نفذت الكمية</div>',
            class: 'stock-out'
        };
    } else if (stock < 10) {
        return {
            html: `<div class="stock-badge stock-low"><i class="fas fa-exclamation"></i> متبقي ${stock}</div>`,
            class: 'stock-low'
        };
    }
    return { html: '', class: '' };
}

// الحصول على أيقونة المنتج
function getProductIcon(category, isDigital) {
    const icons = {
        electronics: 'fas fa-mobile-alt',
        fashion: 'fas fa-tshirt',
        courses: 'fas fa-graduation-cap',
        books: 'fas fa-book',
        default: isDigital ? 'fas fa-download' : 'fas fa-box'
    };
    return icons[category] || icons.default;
}

// الحصول على اسم الفئة
function getCategoryName(category) {
    const names = {
        electronics: 'إلكترونيات',
        fashion: 'أزياء',
        courses: 'دورات',
        books: 'كتب'
    };
    return names[category] || category;
}

// تنسيق السعر
function formatPrice(price) {
    return '$' + parseFloat(price).toLocaleString('en-US', {
        minimumFractionDigits: 2,
        maximumFractionDigits: 2
    });
}

// فتح صفحة تفاصيل المنتج
function openProductDetails(productId) {
    window.location.href = `product-details.php?id=${productId}`;
}

// عرض رسالة خطأ
function showError(message) {
    const grid = document.getElementById('productsGrid');
    grid.innerHTML = `
        <div class="empty-state" style="grid-column: 1/-1;">
            <i class="fas fa-exclamation-triangle" style="color: #ff6b6b;"></i>
            <h3>حدث خطأ</h3>
            <p>${message}</p>
        </div>
    `;
}
