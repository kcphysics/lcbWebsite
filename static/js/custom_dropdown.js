document.addEventListener('DOMContentLoaded', function() {
    var dropdownToggle = document.getElementById('navbarDropdown');
    console.log("Dropdown toggle element (direct):", dropdownToggle); // Added for debugging
    if (dropdownToggle) {
        dropdownToggle.addEventListener('click', function(event) {
            console.log("Dropdown toggle clicked (direct listener)!"); // Added for debugging
            event.preventDefault(); // Prevent default link behavior
            event.stopPropagation(); // Stop event from bubbling up to close other dropdowns

            var dropdownMenu = this.nextElementSibling; // Get the ul.dropdown-menu
            if (dropdownMenu && dropdownMenu.classList.contains('dropdown-menu')) {
                dropdownMenu.classList.toggle('show');
            }

            // Close other open dropdowns
            document.querySelectorAll('.dropdown-menu.show').forEach(function(openMenu) {
                if (openMenu !== dropdownMenu) {
                    openMenu.classList.remove('show');
                }
            });
        });

        // Close the dropdown if the user clicks outside of it
        document.addEventListener('click', function(event) {
            if (!dropdownToggle.contains(event.target) && !dropdownToggle.nextElementSibling.contains(event.target)) {
                var openDropdown = dropdownToggle.nextElementSibling;
                if (openDropdown && openDropdown.classList.contains('show')) {
                    openDropdown.classList.remove('show');
                }
            }
        });
    }
});
