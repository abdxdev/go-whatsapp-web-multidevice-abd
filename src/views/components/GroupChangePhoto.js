export default {
    name: 'GroupChangePhoto',
    data() {
        return {
            loading: false,
            group: '',
            preview_url: null,
            selected_file: null
        }
    },
    computed: {
        group_id() {
            return `${this.group}${window.TYPEGROUP}`;
        }
    },
    methods: {
        openModal() {
            $('#modalGroupChangePhoto').modal({
                onApprove: function () {
                    return false;
                }
            }).modal('show');
        },
        isValidForm() {
            return this.group.trim() !== '' && this.selected_file !== null;
        },
        async handleSubmit() {
            if (!this.isValidForm() || this.loading) {
                return;
            }

            try {
                let response = await this.submitApi();
                showSuccessInfo(response);
                $('#modalGroupChangePhoto').modal('hide');
            } catch (err) {
                showErrorInfo(err);
            }
        },
        async submitApi() {
            this.loading = true;
            try {
                let payload = new FormData();
                payload.append('group_id', this.group_id);
                payload.append('photo', $("#file_group_photo")[0].files[0]);

                let response = await window.http.post(`/group/photo`, payload);
                this.handleReset();
                return response.data.message;
            } catch (error) {
                if (error.response) {
                    throw new Error(error.response.data.message);
                }
                throw new Error(error.message);
            } finally {
                this.loading = false;
            }
        },
        handleReset() {
            this.group = '';
            this.preview_url = null;
            this.selected_file = null;
            $("#file_group_photo").val('');
        },
        handleImageChange(event) {
            const file = event.target.files[0];
            if (file) {
                this.preview_url = URL.createObjectURL(file);
                this.selected_file = file.name;
            }
        }
    },
    template: `
    <div class="green card" @click="openModal" style="cursor:pointer;">
        <div class="content">
            <a class="ui green right ribbon label">Group</a>
            <div class="header">Change Group Photo</div>
            <div class="description">
                Update a group's profile picture
            </div>
        </div>
    </div>
    
    <!--  Modal Change Group Photo  -->
    <div class="ui small modal" id="modalGroupChangePhoto">
        <i class="close icon"></i>
        <div class="header">
            Change Group Photo
        </div>
        <div class="content" style="max-height: 70vh; overflow-y: auto;">
            <div class="ui warning message">
                <i class="info circle icon"></i>
                Please upload a square image (1:1 aspect ratio) to avoid cropping.
                For best results, use an image at least 400x400 pixels.
            </div>
            
            <form class="ui form">
                <div class="field">
                    <label>Group ID</label>
                    <input v-model="group" type="text"
                           placeholder="12036322888236XXXX..."
                           aria-label="Group ID">
                    <input :value="group_id" disabled aria-label="whatsapp_id">
                </div>
                
                <div class="field" style="padding-bottom: 30px">
                    <label>Group Photo</label>
                    <input type="file" style="display: none" id="file_group_photo" accept="image/png,image/jpg,image/jpeg" @change="handleImageChange"/>
                    <label for="file_group_photo" class="ui positive medium green left floated button" style="color: white">
                        <i class="ui upload icon"></i>
                        Upload image
                    </label>
                    <div v-if="preview_url" style="margin-top: 60px">
                        <img :src="preview_url" style="max-width: 100%; max-height: 300px; object-fit: contain" />
                    </div>
                </div>
            </form>
        </div>
        <div class="actions">
            <button class="ui approve positive right labeled icon button" 
                 :class="{'loading': this.loading, 'disabled': !isValidForm() || loading}"
                 @click.prevent="handleSubmit">
                Update Group Photo
                <i class="save icon"></i>
            </button>
        </div>
    </div>
    `
}