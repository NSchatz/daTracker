package com.nschatz.tracker.ui.circle

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Toast
import androidx.fragment.app.Fragment
import androidx.lifecycle.lifecycleScope
import androidx.recyclerview.widget.DividerItemDecoration
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.RecyclerView
import com.nschatz.tracker.data.api.ApiClient
import com.nschatz.tracker.data.model.CircleMember
import com.nschatz.tracker.data.prefs.SessionManager
import com.nschatz.tracker.data.repository.CircleRepository
import com.nschatz.tracker.databinding.FragmentCircleBinding
import com.nschatz.tracker.databinding.ItemMemberBinding
import kotlinx.coroutines.launch

class CircleFragment : Fragment() {

    private var _binding: FragmentCircleBinding? = null
    private val binding get() = _binding!!

    private lateinit var session: SessionManager
    private lateinit var circleRepo: CircleRepository

    private val members: MutableList<CircleMember> = mutableListOf()
    private lateinit var memberAdapter: MemberAdapter

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View {
        _binding = FragmentCircleBinding.inflate(inflater, container, false)
        return binding.root
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)

        session = SessionManager(requireContext())
        circleRepo = CircleRepository(ApiClient(session))

        memberAdapter = MemberAdapter(members)
        binding.recyclerMembers.layoutManager = LinearLayoutManager(requireContext())
        binding.recyclerMembers.addItemDecoration(
            DividerItemDecoration(requireContext(), DividerItemDecoration.VERTICAL)
        )
        binding.recyclerMembers.adapter = memberAdapter

        binding.btnCopyCode.setOnClickListener {
            val code = binding.txtInviteCode.text?.toString() ?: return@setOnClickListener
            val clipboard = requireContext().getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
            clipboard.setPrimaryClip(ClipData.newPlainText("Invite Code", code))
            Toast.makeText(requireContext(), "Invite code copied", Toast.LENGTH_SHORT).show()
        }

        loadCircle()
    }

    private fun loadCircle() {
        viewLifecycleOwner.lifecycleScope.launch {
            circleRepo.getAll().onSuccess { circles ->
                if (circles.isEmpty()) return@onSuccess
                val circle = circles.first()
                session.activeCircleId = circle.id

                binding.txtCircleName.text = circle.name
                binding.txtInviteCode.text = circle.inviteCode

                loadMembers(circle.id)
            }
        }
    }

    private fun loadMembers(circleId: String) {
        viewLifecycleOwner.lifecycleScope.launch {
            circleRepo.getMembers(circleId).onSuccess { list ->
                members.clear()
                members.addAll(list)
                memberAdapter.notifyDataSetChanged()
            }
        }
    }

    override fun onDestroyView() {
        super.onDestroyView()
        _binding = null
    }

    inner class MemberAdapter(
        private val items: List<CircleMember>
    ) : RecyclerView.Adapter<MemberAdapter.ViewHolder>() {

        inner class ViewHolder(val binding: ItemMemberBinding) :
            RecyclerView.ViewHolder(binding.root)

        override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): ViewHolder {
            val b = ItemMemberBinding.inflate(
                LayoutInflater.from(parent.context), parent, false
            )
            return ViewHolder(b)
        }

        override fun onBindViewHolder(holder: ViewHolder, position: Int) {
            val item = items[position]
            holder.binding.txtMemberName.text = item.displayName
            holder.binding.txtMemberRole.text = item.role
        }

        override fun getItemCount() = items.size
    }
}
